package storaged

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os/user"
	"strings"
)

func (s *Server) handleUpdateFolder(writer http.ResponseWriter, req *http.Request) {
	var updateReq UpdateRequest
	submitter, ok := s.readRequest(writer, req, &updateReq)
	if !ok {
		return
	}
	if _, ok := s.Tiers[updateReq.Tier]; !ok {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(writer, "Invalid tier requested: %q does not exist!\n", updateReq.Tier)
		_, _ = fmt.Fprintln(writer, "Check your allocated quota first.")
		return
	}
	err := ValidateProjectName(updateReq.Name)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(writer, "Invalid name requested: %s\n", err)
		return
	}
	sizeInBytes := updateReq.SizeInGB * 1000 * 1000 * 1000
	if sizeInBytes < 0 || sizeInBytes < updateReq.SizeInGB {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintln(writer, "Provided folder size is invalid.")
		return
	}
	status, output := s.attemptAssign(submitter, updateReq)
	if status == http.StatusInternalServerError {
		output += "\n\nTry again later and contact administrators if the folder is in an unexpected state."
	}
	writer.WriteHeader(status)
	_, _ = fmt.Fprintln(writer, output)
}

func (s *Server) attemptAssign(submitter *user.User, updateReq UpdateRequest) (statusCode int, output string) {
	// Check allowed quota.
	allQuota, err := s.allowedQuota(submitter)
	if err != nil {
		return http.StatusInternalServerError, "Failed to calculate quota allocated to user: " + err.Error()
	}
	tierQuota := allQuota[updateReq.Tier]
	quotaFS := s.Tiers[updateReq.Tier]
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	_, quotaUsed, err := QuotaUsed(quotaFS, submitter.Username)
	if err != nil {
		return http.StatusInternalServerError, "Failed to calculate quota used by user: " + err.Error()
	}
	remainingQuota := tierQuota - quotaUsed
	currentQuota, err := quotaFS.Quota(updateReq.Name)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		// Check that the folder does not exist in the main project FS.
		_, err := fs.Stat(s.ProjectFS, updateReq.Name)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			// This is fine. We will create the folder below.
			break
		case err == nil, strings.Contains(err.Error(), "path escapes"):
			return http.StatusBadRequest, "Folder " + updateReq.Name + " already exists in another tier."
		default:
			return http.StatusInternalServerError, "Failed to check project folder existence: " + err.Error()
		}
		currentQuota = 0
	case err != nil:
		return http.StatusInternalServerError, "Failed to calculate quota for existing folder: " + err.Error()
	default:
		// Do a check that the folder actually belongs to us.
		currentOwner, err := quotaFS.FileOwner(updateReq.Name)
		if err != nil {
			return http.StatusInternalServerError, "Failed to fetch owner for existing folder: " + err.Error()
		}
		if currentOwner != submitter.Username {
			return http.StatusBadRequest, "The folder to update does not belong to you!"
		}
	}
	quotaRequested := updateReq.SizeInGB * 1000 * 1000 * 1000
	// Three cases: We are growing storage, shrinking it or doing nothing.
	switch {
	case currentQuota == 0 && quotaRequested == 0:
		return http.StatusOK, "Folder already does not exist."
	case currentQuota == quotaRequested:
		// Doing nothing.
		return http.StatusOK, "Quota is unchanged."
	case currentQuota < quotaRequested:
		// Growing storage.
		quotaNeeded := quotaRequested - currentQuota
		if remainingQuota < quotaNeeded {
			return http.StatusBadRequest, fmt.Sprintf(
				"You do not have sufficient quota left to assign to this tier.\n"+
					"You used %s/%s and have %s left.\n"+
					"This operation needs %s.",
				FormatByteSize(quotaUsed), FormatByteSize(tierQuota), FormatByteSize(remainingQuota),
				FormatByteSize(quotaRequested),
			)
		}
	default:
		// Shrinking storage.
		currentUsage, err := quotaFS.Usage(updateReq.Name)
		if err != nil {
			return http.StatusInternalServerError, "Failed to calculate usage for existing folder: " + err.Error()
		}
		if currentUsage > quotaRequested {
			return http.StatusBadRequest, fmt.Sprintf(
				"You are currently using more storage than the quota you requested.\n"+
					"You are currently using %s.\n"+
					"Please delete some files before requesting to shrink the folder quota.",
				FormatByteSize(currentUsage),
			)
		}
	}
	// We have validated that the operation is valid. Do it now.
	if quotaRequested == 0 {
		// Delete the folder. We have established ownership above.
		err := quotaFS.DeleteFolder(updateReq.Name)
		switch {
		case err != nil && strings.Contains(err.Error(), "directory not empty"):
			return http.StatusBadRequest, "Your directory is not empty."
		case err != nil:
			return http.StatusInternalServerError, fmt.Sprintf(
				"Failed to delete folder: %s", err,
			)
		}
		err = s.ProjectFS.DeleteLink(updateReq.Name)
		if err != nil {
			return http.StatusInternalServerError, fmt.Sprintf(
				"Failed to delete link: %s", err,
			)
		}
		return http.StatusOK, "Your project folder has been deleted."
	}
	if currentQuota == 0 {
		// If the folder did not exist previously, create it.
		err := quotaFS.CreateFolder(updateReq.Name, submitter.Uid, submitter.Gid)
		if err != nil {
			return http.StatusInternalServerError, fmt.Sprintf(
				"Failed to create folder: %s", err,
			)
		}
	}
	err = quotaFS.SetQuota(updateReq.Name, quotaRequested)
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf(
			"Failed to create folder: %s", err,
		)
	}
	if currentQuota != 0 {
		return http.StatusOK, "Your folder's quota has been updated."
	}
	// We need to create the symlink as well.
	err = s.ProjectFS.CreateLink(updateReq.Name, quotaFS.PathFor(updateReq.Name))
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf(
			"Failed to create symlink for folder: %s", err,
		)
	}
	return http.StatusOK, fmt.Sprintf(
		"Your folder has been created.\n"+
			"You can access it at %s.\n",
		s.ProjectFS.PathFor(updateReq.Name),
	)
}
