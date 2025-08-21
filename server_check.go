package storaged

import (
	"fmt"
	"net/http"
	"os/user"
	"slices"
	"strings"
)

const MaxDisplayedFolderPerTier = 5

func (s *Server) handleCheckQuota(writer http.ResponseWriter, req *http.Request) {
	var checkReq CheckQuotaRequest
	submitter, ok := s.readRequest(writer, req, &checkReq)
	if !ok {
		return
	}
	checkTarget := submitter
	if checkTarget.Username != checkReq.User {
		var err error
		checkTarget, err = user.Lookup(checkReq.User)
		if err != nil {
			http.Error(writer, "Cannot find requested user: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	allowed, err := s.allowedQuota(checkTarget)
	if err != nil {
		http.Error(
			writer,
			"Failed to calculate quota allocated to user: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}
	outputEntries := make([]quotaEntry, 0, len(s.Tiers))
	folderOmitted := false
	for tierName, quotaFS := range s.Tiers {
		entries, usedQuota, err := QuotaUsed(quotaFS, checkReq.User)
		if err != nil {
			http.Error(
				writer,
				"Failed to retrieve quota used by user in "+tierName+": "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
		if usedQuota == 0 && allowed[tierName] == 0 {
			// This directory is not interesting. Don't even bother outputting it to the user.
			continue
		}
		slices.SortStableFunc(entries, func(a, b Quota) int {
			return a.Quota - b.Quota
		})
		if len(entries) > MaxDisplayedFolderPerTier {
			folderOmitted = true
			entries = entries[:MaxDisplayedFolderPerTier]
		}
		outputEntries = append(outputEntries, quotaEntry{
			Name:         tierName,
			UsageEntries: entries,
			UsedQuota:    usedQuota,
			AllowedQuota: allowed[tierName],
		})
	}
	slices.SortFunc(outputEntries, func(a, b quotaEntry) int {
		return strings.Compare(a.Name, b.Name)
	})
	if len(outputEntries) == 0 {
		_, _ = fmt.Fprintf(writer, "User %s has no access to managed storage.\n", checkReq.User)
		return
	}
	_, _ = fmt.Fprintf(writer, "User %s has access to the following tiers of storage:\n\n", checkReq.User)
	for _, v := range outputEntries {
		_, _ = fmt.Fprintf(
			writer,
			"%s - %s assigned / %s allocated\n",
			v.Name, FormatByteSize(v.UsedQuota), FormatByteSize(v.AllowedQuota),
		)
		for _, w := range v.UsageEntries {
			_, _ = fmt.Fprintf(
				writer,
				"\t%s - %s used / %s assigned\n",
				w.Name, FormatByteSize(w.Usage), FormatByteSize(w.Quota),
			)
		}
	}
	if folderOmitted {
		_, _ = fmt.Fprintln(writer, "\nNote that the smaller folders have been omitted for brevity.")
	}
}

type quotaEntry struct {
	Name         string
	UsageEntries []Quota
	UsedQuota    int
	AllowedQuota int
}
