package refresh

import (
	"os"
	"path"

	homedir "github.com/mitchellh/go-homedir"
)

var LogLocation = func() string {
	dir, _ := homedir.Dir()
	dir, _ = homedir.Expand(dir)
	dir = path.Join(dir, ".refresh")
	os.MkdirAll(dir, 0755)
	return dir
}

var ErrorLogPath = func() string {
	return path.Join(LogLocation(), ID()+".err")
}
