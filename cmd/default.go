/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-10 01:25
**/

package cmd

func Default(args []string) string {
	switch args[0] {
	case "switch":
		return Switch(args)
	case "recover":
		return Recover(args)
	default:
		return Help()
	}
}
