package seednode

const defaultConfigTemplate = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

# NOTE: Any path below can be absolute (e.g. "/var/myawesomeapp/data") or
# relative to the home directory (e.g. "data"). The home directory is
# "$HOME/.tendermint" by default, but could be changed via $TMHOME env variable
# or --home cmd flag.

#######################################################
###     Multiseed Server Configuration Options      ###
#######################################################
# Port for the frontend
http_port = "{{ .HttpPort }}"

# Output level for logging: "none", info", "error", "debug". debug will enable pex and addrbook (very) verbose logs
log_level = "{{ .LogLevel }}"

# Chain specific config
[terra]
chain_id = "{{ .Terra.ChainId }}"
seeds = "{{ .Terra.Seeds }}"
laddr = "{{ .Terra.ListenAddress }}"

[bombay]
chain_id = "{{ .Bombay.ChainId }}"
seeds = "{{ .Bombay.Seeds }}"
laddr = "{{ .Bombay.ListenAddress }}"

[band]
chain_id = "{{ .Band.ChainId }}"
seeds = "{{ .Band.Seeds }}"
laddr = "{{ .Band.ListenAddress }}"

[fetchai]
chain_id = "{{ .Fetchai.ChainId }}"
seeds = "{{ .Fetchai.Seeds }}"
laddr = "{{ .Fetchai.ListenAddress }}"

[injective]
chain_id = "{{ .Injective.ChainId }}"
seeds = "{{ .Injective.Seeds }}"
laddr = "{{ .Injective.ListenAddress }}"

[persistence]
chain_id = "{{ .Persistence.ChainId }}"
seeds = "{{ .Persistence.Seeds }}"
laddr = "{{ .Persistence.ListenAddress }}"

[irisnet]
chain_id = "{{ .Irisnet.ChainId }}"
seeds = "{{ .Irisnet.Seeds }}"
laddr = "{{ .Irisnet.ListenAddress }}"

[sentinel]
chain_id = "{{ .Sentinel.ChainId }}"
seeds = "{{ .Sentinel.Seeds }}"
laddr = "{{ .Sentinel.ListenAddress }}"

[certik]
chain_id = "{{ .Certik.ChainId }}"
seeds = "{{ .Certik.Seeds }}"
laddr = "{{ .Certik.ListenAddress }}"

[lum]
chain_id = "{{ .Lum.ChainId }}"
seeds = "{{ .Lum.Seeds }}"
laddr = "{{ .Lum.ListenAddress }}"

[sifchain]
chain_id = "{{ .Sifchain.ChainId }}"
seeds = "{{ .Sifchain.Seeds }}"
laddr = "{{ .Sifchain.ListenAddress }}"

[desmos]
chain_id = "{{ .Desmos.ChainId }}"
seeds = "{{ .Desmos.Seeds }}"
laddr = "{{ .Desmos.ListenAddress }}"

`
