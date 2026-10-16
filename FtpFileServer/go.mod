module ftpfileserver

go 1.20

require github.com/goftp/server v0.0.0-20200708154336-f64f7c2d8a42

require (
	github.com/goftp/file-driver v0.0.0-20180502053751-5d604a0fc0c9 // indirect
	github.com/jlaffaye/ftp v0.2.2 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
)

replace github.com/goftp/server => ./third_party/server
