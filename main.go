package main

func main() {
	config, err := LoadConfig()
	if err != nil {
		panic(err)
	}

	client, err := NewBunqClient(config)
	if err != nil {
		panic(err)
	}
	if err := client.LoadInstallation(); err != nil {
		panic(err)
	}
	if err := client.LoadDeviceServer(); err != nil {
		panic(err)
	}
}
