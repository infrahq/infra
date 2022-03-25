package cmd

var loginExample = `
#By default, login will prompt for all required information. 
$ infra login 

#Login to a specified server
$ infra login SERVER
$ infra login --server SERVER

#Login with an access key 
$ infra login --key KEY 

#Login with a specified provider
$ infra login --provider NAME

#Use the '--non-interactive' flag to error out instead of prompting. 
`
