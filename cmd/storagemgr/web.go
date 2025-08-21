package main

import "github.com/NTUEEECluster/storaged"

func newQuotaRequest(hostURL string, lookupTarget string) webRequestModel {
	return NewWebRequestModel("Loading quota information...", hostURL+"/quota", storaged.CheckQuotaRequest{
		User: lookupTarget,
	})
}

func newUpdateRequest(hostURL string, projectName string, projectTier string, sizeInGB int) webRequestModel {
	return NewWebRequestModel("Requesting server to update quota allocation...", hostURL+"/folders", storaged.UpdateRequest{
		Name:     projectName,
		Tier:     projectTier,
		SizeInGB: sizeInGB,
	})
}
