package storaged

import (
	"errors"
	"fmt"
	"os/user"
	"strings"

	"github.com/ccin2p3/go-freeipa/freeipa"
)

type IPAClientConfig struct {
	// GroupPrefix is the prefix of the group that needs to match before any request is allowed to proceed.
	GroupPrefix string

	// Host is the address of the FreeIPA host.
	Host string
	// Username is the name of the user with permission to do the necessary group modifications.
	Username string
	// Password is the password of the user with permission to do the necessary group modifications.
	Password string
}

type IPAClient struct {
	IPAClientConfig
	cli *freeipa.Client
}

func NewIPAClient(cfg IPAClientConfig) (*IPAClient, error) {
	cli, err := freeipa.Connect(cfg.Host, nil, cfg.Username, cfg.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to FreeIPA host: %w", err)
	}
	return &IPAClient{
		IPAClientConfig: cfg,
		cli:             cli,
	}, nil
}

func (cli *IPAClient) GroupAdd(groupName string) error {
	err := validateGroupAbsent(groupName, cli.GroupPrefix)
	if err != nil {
		return fmt.Errorf("error validating group name: %w", err)
	}
	description := "Automated group created by storaged"
	_, err = cli.cli.GroupAdd(&freeipa.GroupAddArgs{Cn: groupName}, &freeipa.GroupAddOptionalArgs{
		Description: &description,
	})
	if err != nil {
		return fmt.Errorf("error creating group: %w", err)
	}
	return nil
}

func (cli *IPAClient) GroupRemove(groupName string) error {
	err := validateGroupExist(groupName, cli.GroupPrefix)
	if err != nil {
		return fmt.Errorf("error validating group name: %w", err)
	}
	_, err = cli.cli.GroupDel(
		&freeipa.GroupDelArgs{Cn: []string{groupName}},
		&freeipa.GroupDelOptionalArgs{},
	)
	if err != nil {
		return fmt.Errorf("error deleting group: %w", err)
	}
	return nil
}

func (cli *IPAClient) GroupMembers(groupName string, limit int) ([]string, error) {
	err := validateGroupExist(groupName, cli.GroupPrefix)
	if err != nil {
		return nil, fmt.Errorf("error validating group name: %w", err)
	}
	groupNames := []string{groupName}
	userFind, err := cli.cli.UserFind("", &freeipa.UserFindArgs{}, &freeipa.UserFindOptionalArgs{
		InGroup:   &groupNames,
		Sizelimit: &limit,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing existing users in group: %w", err)
	}
	result := make([]string, len(userFind.Result))
	for i := range userFind.Result {
		result[i] = userFind.Result[i].UID
	}
	return result, nil
}

func (cli *IPAClient) GroupMemberAdd(groupName string, member string) error {
	err := validateGroupExist(groupName, cli.GroupPrefix)
	if err != nil {
		return err
	}
	err = validateUserExist(member)
	if err != nil {
		return err
	}
	userToAdd := []string{member}
	_, err = cli.cli.GroupAddMember(
		&freeipa.GroupAddMemberArgs{
			Cn: groupName,
		}, &freeipa.GroupAddMemberOptionalArgs{
			User: &userToAdd,
		},
	)
	if err != nil {
		return fmt.Errorf("error adding member to group: %w", err)
	}
	return nil
}

func (cli *IPAClient) GroupMemberRemove(groupName string, member string) error {
	err := validateGroupExist(groupName, cli.GroupPrefix)
	if err != nil {
		return err
	}
	err = validateUserExist(member)
	if err != nil {
		return err
	}
	userToAdd := []string{member}
	_, err = cli.cli.GroupRemoveMember(
		&freeipa.GroupRemoveMemberArgs{
			Cn: groupName,
		}, &freeipa.GroupRemoveMemberOptionalArgs{
			User: &userToAdd,
		},
	)
	if err != nil {
		return fmt.Errorf("error removing member from group: %w", err)
	}
	return nil
}

func validateUserExist(userLogin string) error {
	_, err := user.Lookup(userLogin)
	if err != nil {
		return fmt.Errorf("error validating user %q", userLogin)
	}
	return nil
}

func validateGroupExist(groupName string, safePrefix string) error {
	err := validateGroupName(groupName, safePrefix)
	if err != nil {
		return err
	}
	_, err = user.LookupGroup(groupName)
	if err != nil {
		return fmt.Errorf("error validating group %q exists", groupName)
	}
	return nil
}

func validateGroupAbsent(groupName string, safePrefix string) error {
	err := validateGroupName(groupName, safePrefix)
	if err != nil {
		return err
	}
	_, err = user.LookupGroup(groupName)
	if err == nil {
		return fmt.Errorf("group %q already exists", groupName)
	}
	return nil
}

func ValidateProjectName(projectName string) error {
	if len(projectName) < 3 || len(projectName) > 20 {
		return errors.New("project name must be between 3 and 20 characters long")
	}
	if !isAlphanumeric(projectName) {
		return errors.New("project name contains unsafe characters")
	}
	return nil
}

func validateGroupName(groupName string, safePrefix string) error {
	if !strings.HasPrefix(groupName, safePrefix) {
		return fmt.Errorf("group name %q does not have expected prefix %q", groupName, safePrefix)
	}
	groupName = strings.TrimPrefix(groupName, safePrefix)
	if len(groupName) < 3 || len(groupName) > 20 {
		return errors.New("group name must be between 3 and 20 characters long")
	}
	if !isAlphanumeric(groupName) {
		return errors.New("group name contains unsafe characters")
	}
	return nil
}

func isAlphanumeric(name string) bool {
	for _, v := range name {
		switch {
		case v >= 'A' && v <= 'Z':
		case v >= 'a' && v <= 'z':
		case v >= '0' && v <= '9':
		default:
			return false
		}
	}
	return true
}
