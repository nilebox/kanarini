set -euo pipefail
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
cd $DIR

rm apiserver.csr
rm apiserver.pem
rm apiserver-key.pem
rm cm-adapter-serving-certs.yaml
rm metrics-ca.crt
rm metrics-ca.key
rm metrics-ca-config.json