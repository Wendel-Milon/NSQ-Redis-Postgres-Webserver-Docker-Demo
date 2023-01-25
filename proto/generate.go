package proto

// means to following:
//		-I= source folder
// 		--go_out= where to put the go files. Two points so that the compiler does not create a subfolder
// 		path to the file.
//go:generate protoc -I=. --go_out=.. ./addressbook.proto

// Does not work....
//go:generate protoc -I=. --go_out=.. --go-grpc-out=.. ./greeter.proto
