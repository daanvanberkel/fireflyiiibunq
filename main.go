package main

func main() {
	client := NewBunqClient()
	if err := client.LoadInstallation(); err != nil {
		panic(err)
	}
}
