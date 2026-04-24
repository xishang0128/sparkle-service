//go:build windows

package core

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func secureCoreBinary(corePath string) error {
	if os.Getenv("SPARKLE_SKIP_CORE_ACL_HARDENING") == "1" {
		return nil
	}

	if err := setRestrictedACL(filepath.Dir(corePath), windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT); err != nil {
		return fmt.Errorf("加固核心目录权限失败：%w", err)
	}
	if err := setRestrictedACL(corePath, windows.NO_INHERITANCE); err != nil {
		return fmt.Errorf("加固核心文件权限失败：%w", err)
	}

	return nil
}

func setRestrictedACL(path string, inheritance uint32) error {
	currentSID, err := currentProcessSID()
	if err != nil {
		return err
	}
	systemSID, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return err
	}
	adminSID, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return err
	}

	acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{
		fullAccessEntry(currentSID, windows.TRUSTEE_IS_USER, inheritance),
		fullAccessEntry(systemSID, windows.TRUSTEE_IS_USER, inheritance),
		fullAccessEntry(adminSID, windows.TRUSTEE_IS_GROUP, inheritance),
	}, nil)
	if err != nil {
		return err
	}

	return windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		acl,
		nil,
	)
}

func fullAccessEntry(sid *windows.SID, trusteeType windows.TRUSTEE_TYPE, inheritance uint32) windows.EXPLICIT_ACCESS {
	return windows.EXPLICIT_ACCESS{
		AccessPermissions: windows.GENERIC_ALL,
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       inheritance,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  trusteeType,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}
}

func currentProcessSID() (*windows.SID, error) {
	token := windows.GetCurrentProcessToken()
	user, err := token.GetTokenUser()
	if err != nil {
		return nil, fmt.Errorf("读取当前进程 SID 失败：%w", err)
	}

	return user.User.Sid, nil
}
