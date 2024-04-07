package Build

import (
	"log"
	"os"
	"os/exec"
	"time"
)

func CheckArgs() {
	os.Setenv("GOOS", "windows")
	os.Setenv("GOARCH", "amd64")

	if (len(os.Args) == 2) {
		if (os.Args[1] == "install") {
			InstallNode()
		} else if (os.Args[1] == "build") {
			BuildReact()
		}
	}

	if (len(os.Args) == 3) {
		if ((os.Args[1] == "install" && os.Args[2] == "build") || (os.Args[2] == "install" && os.Args[1] == "build")) {
			InstallNode()
			BuildReact()
		}
	}
}

func InstallNode() {
	err := os.Chdir("./Frontend")
	if err != nil {log.Fatal(err)}

	cmd := exec.Command("npm", "install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {log.Fatal(err)}

	err = os.Chdir("..")
	if err != nil {log.Fatal(err)}

	for i := 0; i < 5; i++ {
		time.Sleep(250 * time.Millisecond)
		log.Println("waiting for files...")
	}

	log.Println("Node packages installed.")
}

func BuildReact() {
	err := os.Chdir("./Frontend")
	if err != nil {log.Fatal(err)}

	cmd := exec.Command("npm", "run", "build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {log.Fatal(err)}

	err = os.Chdir("..")
	if err != nil {log.Fatal(err)}

	for i := 0; i < 5; i++ {
		time.Sleep(250 * time.Millisecond)
		log.Println("waiting for files...")
	}

	log.Println("React project build complete.")
}
