package core

type fileAccess struct {
	groupID int
	ok      bool
}

type LaunchOption func(*launchOptions)

type launchOptions struct {
	fileAccess fileAccess
}

func WithLogFileGroup(groupID uint32) LaunchOption {
	return func(options *launchOptions) {
		options.fileAccess = fileAccess{
			groupID: int(groupID),
			ok:      true,
		}
	}
}

func withFileAccess(access fileAccess) LaunchOption {
	return func(options *launchOptions) {
		options.fileAccess = access
	}
}

func collectLaunchOptions(options []LaunchOption) launchOptions {
	var collected launchOptions
	for _, option := range options {
		if option != nil {
			option(&collected)
		}
	}
	return collected
}
