# lib3d_gl – Makefile

BINARY := lib3d_gl.app

.PHONY: dev build clean

dev:
	go run .

build:
	go build -o $(BINARY) .

clean:
	rm -f $(BINARY)
