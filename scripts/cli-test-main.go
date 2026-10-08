//go:build ignore

// A test-only entry point into the exact pinned CLI command package. All HTTP
// traffic goes to a local mock; production dashboard imports no CLI commands.
package main

import (
 "os"
 "github.com/cronitorio/cronitor-cli/cmd"
 "github.com/cronitorio/cronitor-cli/lib"
)

func main() {
 lib.BaseURLOverride = os.Getenv("DASHBOARD_TEST_API")
 lib.PingHostOverride = os.Getenv("DASHBOARD_TEST_API")
 // Match the released CLI entry point's exec argument separator behavior.
 for i,arg := range os.Args {
  if arg == "exec" && len(os.Args)>i+2 {
   codeIndex := i+1
   if os.Args[codeIndex]=="--no-stdout" { codeIndex++ }
   position := codeIndex+1
   os.Args=append(os.Args,"")
   copy(os.Args[position+1:],os.Args[position:])
   os.Args[position]="--"
   break
  }
 }
 cmd.Execute()
}
