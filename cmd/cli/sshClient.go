package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"time"

	"golang.org/x/crypto/ssh"
)

// https://subscription.packtpub.com/book/networking-&-servers/9781788627917/7/ch07lvl1sec53/using-the-go-ssh-client

func GetSSHConfig(host, user, pwd string) (conf *ssh.ClientConfig) {

	// pKey := []byte("<privateKey>") // TODO

	// var err error
	//var signer ssh.Signer

	// signer, err = ssh.ParsePrivateKey(pKey)
	// if err != nil {
	// 	fmt.Println(err.Error())
	// }

	//The HostKeyCallback is required as a means for the client to verify the identity of the host and prevent the Man In The Middle attack
	// This however can be bypassed by passing the InsecureIgnoreHostKey function which will allow any host key to be used, but this should not be used in production code.

	// var hostkeyCallback ssh.HostKeyCallback
	// hostkeyCallback, err = knownhosts.New("~/.ssh/known_hosts")
	// if err != nil {
	// 	fmt.Println(err.Error())
	// }

	conf = &ssh.ClientConfig{
		User: user,
		//HostKeyCallback: hostkeyCallback,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password(pwd),
			//ssh.PublicKeys(signer),
		},
	}
	return

}

func GetSSHClient(host, user, pwd string) {

	var client *ssh.Client

	client, err := ssh.Dial("tcp", host, GetSSHConfig(host, user, pwd))
	if err != nil {
		fmt.Println(err.Error())
	}
	defer client.Close()

	var session *ssh.Session
	var stdin io.WriteCloser
	var stdout, stderr io.Reader

	// start ssh session
	// Multiple sessions per client are allowed
	session, err = client.NewSession()
	if err != nil {
		fmt.Println(err.Error())
	}
	defer session.Close()

	// send stdin
	stdin, err = session.StdinPipe()
	if err != nil {
		fmt.Println(err.Error())
	}

	wr := make(chan []byte, 10)

	go func() {
		for {
			select {
			case d := <-wr:
				_, err := stdin.Write(d)
				if err != nil {
					fmt.Println(err.Error())
				}
			}
		}
	}()

	// get stdout
	stdout, err = session.StdoutPipe()
	if err != nil {
		fmt.Println(err.Error())
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for {
			if tkn := scanner.Scan(); tkn {
				rcv := scanner.Bytes()

				raw := make([]byte, len(rcv))
				copy(raw, rcv)
				fmt.Println("::", string(raw))

				fmt.Println("\n\n:>>>:", EncodeToString(raw))
			} else {
				if scanner.Err() != nil {
					fmt.Println(scanner.Err())
				} else {
					fmt.Println("io.EOF")
				}
				return
			}
		}
	}()

	// get stderr
	stderr, err = session.StderrPipe()
	if err != nil {
		fmt.Println(err.Error())
	}

	go func() {
		scanner := bufio.NewScanner(stderr)

		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	// err = session.Run("cd /qsys.lib/sumitg1.lib")

	// if err != nil {
	// 	log.Fatal("Error executing command 1 . ", err)
	// }

	// ================================================
	// run single command per session
	// ================================================
	//err = session.Run("cd /qsys.lib/sumitg1.lib/TOYOTA.FILE && ls -al *.MBR")
	//err = session.Run("cd /qsys.lib/sumitg1.lib/TOYOTA.FILE && cat TDR00DT.MBR")

	//err = session.Run("cd /qsys.lib/sumitg1.lib && ls -al *.FILE")
	err = session.Run("cd /qsys.lib/sumitg1.lib/TOYOTA.FILE && grep -i TDR00DT.MBR")

	if err != nil {
		log.Fatal("Error executing command 2 . ", err)
	}

	// ================================================
	// run multiple commands per session  START
	// ================================================

	// err = session.Shell()
	// if err != nil {
	// 	log.Fatal("Error session.Shell() . ", err)
	// }
	// stdin.Write([]byte("cd /qsys.lib/sumitg1.lib/TOYOTA.FILE \n"))

	// //session.Wait()
	// // The command has been sent to the device, but you haven't gotten output back yet.
	// // Not that you can't send more commands immediately.
	// stdin.Write([]byte("ls \n"))

	// session.Wait()
	// ================================================
	// run multiple commands per session  END
	// ================================================
	// Then you'll want to wait for

	// session.Shell()

	// for {
	// 	fmt.Println("$")

	// 	scanner := bufio.NewScanner(os.Stdin)
	// 	scanner.Scan()
	// 	text := scanner.Text()

	// 	wr <- []byte(text + "\n")
	// }
	time.Sleep(10 * time.Second)
}
