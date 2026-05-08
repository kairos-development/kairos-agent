#!/bin/bash
set -e

# Clone dependencies into the paths that replace directives expect
rm -rf ../kairos-contracts ../kairos-connectors
git clone --depth 1 https://github.com/kairos-development/kairos-contracts.git ../kairos-contracts
git clone --depth 1 https://github.com/kairos-development/kairos-connectors.git ../kairos-connectors

echo "Dependencies cloned to ../kairos-contracts and ../kairos-connectors"
