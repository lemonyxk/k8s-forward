/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
*
* @create: 2022-02-10 01:25
**/

package cmd

import "os"

func Default(args []string) string {

	os.Args = args

	if len(os.Args) < 4 {
		return Help()
	}

	switch os.Args[1] {
	case "switch":
		return Switch()
	case "recover":
		return Recover()
	default:
		return Help()
	}
}
