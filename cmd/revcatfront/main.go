package main

import "github.com/99designs/gqlgen/client"

func main() {
	c := client.New("http://localhost:8080/graphql")
}
