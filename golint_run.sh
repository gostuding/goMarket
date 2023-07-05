~/go/bin/golangci-lint run -c .golangci.yml > ./golangci-lint/report-unformatted.json
cd ./golangci-lint
python3 ./recomp.py
