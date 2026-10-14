# Policy fixture

`statement.json` contains the statement from the first successful GitHub Actions run.
The test fixture does not include its signature. The CUE CLI uses it to test the ancestry policy.
The public verification job uses Cosign to verify real signatures before it applies that policy.
