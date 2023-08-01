$(go env GOPATH)/bin/golangci-lint run -c ./golangci-lint/.golangci.yml > ./golangci-lint/report-unformatted.json
cd ./golangci-lint
python3 ./recomp.py
rm ./report-unformatted.json
