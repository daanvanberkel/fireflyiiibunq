package main

import "os"

func main() {
	client := NewBunqClient(os.Getenv("BUNQ_API_KEY"))
	if err := client.LoadInstallation(); err != nil {
		panic(err)
	}

	if err := client.LoadDeviceServer(); err != nil {
		panic(err)
	}
}
