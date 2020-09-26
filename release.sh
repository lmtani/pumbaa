for GOOS in darwin linux; do
    echo "Building $GOOS-$GOARCH"
    export GOARCH=amd64
    go build -o bin/cromwell-cli-$GOOS-$GOARCH
done
