package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
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
                <CompanyID>autoresponder.myopenfactory.com</CompanyID>
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
		<Features>
			<Feature>
				<ClassID>CLIENT</ClassID>
				<FeatureID>User</FeatureID>
				<Value>%s</Value>
			</Feature>
		</Features>
	</Body>
	<Subject>#MIRROR</Subject>
    <MessageID>%s</MessageID>
    <ReceiverID>autoresponder.myopenfactory.com</ReceiverID>
    <SenderID>client.myopenfactory.test</SenderID>
    <TypeID>ORDER</TypeID>
</Message>`

func main() {
	basePath := "/tmp/myof"
	if runtime.GOOS == "windows" {
		basePath = "C:/myof"
	}

	messageId := time.Now().Format("20060102150405Z0700")
	attachmentPath := filepath.Join(basePath, "attachment")
	err := os.WriteFile(filepath.Join(attachmentPath, fmt.Sprintf("attachment-%s.sample", messageId)), []byte(fmt.Sprintf("%s", messageId)), 0666)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	outboundPath := filepath.Join(basePath, "outbound")
	err = os.WriteFile(filepath.Join(outboundPath, "message.xml"), []byte(fmt.Sprintf(messageTpl, runtime.GOOS, messageId)), 0666)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	messageId = time.Now().Format("20060102150405Z0700") + "-special"
	attachmentSpecialPath := filepath.Join(basePath, "attachment_special")
	err = os.WriteFile(filepath.Join(attachmentSpecialPath, fmt.Sprintf("attachment-%s.sample", messageId)), []byte(fmt.Sprintf("%s", messageId)), 0666)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	outboundSpecialPath := filepath.Join(basePath, "outbound_special")
	err = os.WriteFile(filepath.Join(outboundSpecialPath, "message"), []byte(fmt.Sprintf(messageTpl, runtime.GOOS+"-special", messageId)), 0666)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	inboundPath := filepath.Join(basePath, "inbound")
	inboundSpecialPath := filepath.Join(basePath, "inbound_special")
	inboundAttachmentPath := filepath.Join(basePath, "attachment_inbound")
	inboundAttachmentSpecialPath := filepath.Join(basePath, "attachment_inbound_special")

	var wg sync.WaitGroup
	wg.Go(func() {
		checkForFile(inboundPath, attachmentPath, inboundAttachmentPath, regexp.MustCompile("^[a-f\\d]{24}$"))
	})
	wg.Go(func() {
		checkForFile(inboundSpecialPath, attachmentSpecialPath, inboundAttachmentSpecialPath, regexp.MustCompile("test"))
	})
	complete := make(chan struct{})
	go func() {
		wg.Wait()
		close(complete)
	}()

	timeout := time.After(5 * time.Minute)
	select {
	case <-timeout:
		fmt.Println("Timed out!")
		os.Exit(1)
		return
	case <-complete:
		return
	}
}

func checkForFile(inboundFolder, attachmentFolder, attachmentInboundFolder string, filenameMatcher *regexp.Regexp) {
	for {
		files, err := os.ReadDir(inboundFolder)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if len(files) > 0 {
			attachments, err := os.ReadDir(attachmentFolder)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if len(attachments) > 0 {
				fmt.Println("Attachment file still present in the directory; expected it to be removed after processing.")
				os.Exit(1)
			}

			for _, file := range files {
				if !filenameMatcher.MatchString(file.Name()) {
					fmt.Printf("File %s does not match expected pattern %s\n", file.Name(), filenameMatcher.String())
					os.Exit(1)
				}
			}

			attachments, err = os.ReadDir(attachmentInboundFolder)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if len(attachments) == 0 {
				fmt.Println("Missing attachment in download folder; expected one to be there.")
				os.Exit(1)
			}
			return
		}
		time.Sleep(time.Second)
	}
}
