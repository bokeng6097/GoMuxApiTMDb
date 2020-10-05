// main.go

package main

func main() {
	a := App{}

	a.Initialize("austin", "123456", "go_mux_api")

	a.Run(":8080")
}
