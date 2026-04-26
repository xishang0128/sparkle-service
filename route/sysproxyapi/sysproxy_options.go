package sysproxyapi

import "github.com/xishang0128/sysproxy-go/sysproxy"

func cloneSysproxyOptions(opt *sysproxy.Options) *sysproxy.Options {
	if opt == nil {
		return &sysproxy.Options{}
	}
	copied := *opt
	return &copied
}
