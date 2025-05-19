package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var messageTpl = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Message xmlns="http://myopenfactory.net/myopenfactory50/">
    <Body>
        <Companies>
            <Company>
                <CompanyID>client.myopenfactory.test</CompanyID>
                <Name>myOpenFactory Client Test</Name>
            </Company>
            <Company>
                <CompanyID>myopenfactory.test</CompanyID>
                <Name>myOpenFactory DevOp Test</Name>
            </Company>
        </Companies>
        <Items>
            <Item>
                <Deliveries>
                    <Delivery>
                        <Quantity>10.0</Quantity>
                    </Delivery>
                </Deliveries>
                <ItemID>1</ItemID>
                <Unit>PCE</Unit>
            </Item>
        </Items>
	</Body>
	<Subject>MIRROR</Subject>
    <MessageID>%s</MessageID>
    <ReceiverID>myopenfactory.test</ReceiverID>
    <SenderID>client.myopenfactory.test</SenderID>
    <TypeID>ORDER</TypeID>
</Message>`

func main() {
	basePath := "/tmp/myof"
	if runtime.GOOS == "windows" {
		basePath = "C:/myof"
	}

	outboundPath := filepath.Join(basePath, "outbound")
	err := os.WriteFile(filepath.Join(outboundPath, "message.xml"), []byte(fmt.Sprintf(messageTpl, time.Now().Format(time.RFC3339))), 0666)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	attachmentPath := filepath.Join(basePath, "attachment")
	err = os.WriteFile(filepath.Join(attachmentPath, "attachment.sample"), []byte(fmt.Sprintf("%s", time.Now().Format(time.RFC3339))), 0666)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	inboundPath := filepath.Join(basePath, "inbound")

	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			fmt.Println("Timed out!")
			return
		case <-ticker.C:
			files, err := os.ReadDir(inboundPath)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if len(files) > 0 {
				attachments, err := os.ReadDir(attachmentPath)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				if len(attachments) > 0 {
					fmt.Println("Attachment file still present in the directory; expected it to be removed after processing.")
					os.Exit(1)
				}

				for _, file := range files {
					log.Println(file.Name())
				}
				return
			}
		}
	}
}
