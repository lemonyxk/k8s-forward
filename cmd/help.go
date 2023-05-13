/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
*
* @create: 2022-02-10 03:23
**/

package cmd

func Help() string {
	return `
Usage: k8s-forward connect
  -- connect to kubernetes cluster and start the server

Usage: k8s-forward clean
  -- clean the resource if terminated unexpectedly

Usage: k8s-forward switch [args]
  -- switch the kubernetes cluster

Usage: k8s-forward recover [args]
  -- switch the kubernetes cluster
`
}
