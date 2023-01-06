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

# Chains specific config
chains:
 - osmosis
   chain_id = "osmosis-1"
   bootstrap-peers = "seeds..."
   laddr = "tcp://0.0.0.0:26656"
 - terra
   chain_id = "phoenix-1"
   bootstrap-peers = "seeds..."
   laddr = "tcp://0.0.0.0:26657"
`
