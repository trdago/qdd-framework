package main
import (
	"fmt"
	"os"
	"gopkg.in/yaml.v3"
)
func main() {
	fnds, _ := os.ReadDir("../.qdd/project/findings")
	for _, f := range fnds {
		data, _ := os.ReadFile("../.qdd/project/findings/" + f.Name())
		var raw map[string]interface{}
		yaml.Unmarshal(data, &raw)
		status := fmt.Sprintf("%v", raw["status"])
		fmt.Println(f.Name(), status)
	}
}
