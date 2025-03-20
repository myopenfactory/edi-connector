package version

import (
	"fmt"

	"github.com/myopenfactory/edi-connector/v2/version"
)

func Run() error {
	fmt.Println("Version:", version.Version)
	fmt.Println("Date:", version.Date)
	fmt.Println("Commit:", version.Commit)
	return nil
}
