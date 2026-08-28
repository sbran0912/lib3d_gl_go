# lib3d_gl – Makefile

BINARY := demo.app

.PHONY: dev build clean

dev:
	go run .

build:
	go build -o $(BINARY) .

clean:
	rm -f $(BINARY)
