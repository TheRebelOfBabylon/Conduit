/*
Copyright (C) 2015-2018 Lightning Labs and The Lightning Network Developers
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package core

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/TheRebelOfBabylon/Conduit/utils"
	flags "github.com/jessevdk/go-flags"
	"github.com/lightningnetwork/lnd/lnrpc/chainrpc"
	"github.com/lightningnetwork/lnd/lnrpc/invoicesrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/lightningnetwork/lnd/lnrpc/watchtowerrpc"
	"github.com/lightningnetwork/lnd/lnrpc/wtclientrpc"
	yaml "gopkg.in/yaml.v2"
)

type subRPCServerConfigs struct {

	// ChainRPC is a sub-RPC server that exposes functionality allowing a
	// client to be notified of certain on-chain events (new blocks,
	// confirmations, spends).
	ChainRPC *chainrpc.Config `group:"chainrpc" namespace:"chainrpc"`

	// InvoicesRPC is a sub-RPC server that exposes invoice related methods
	// as a gRPC service.
	InvoicesRPC *invoicesrpc.Config `group:"invoicesrpc" namespace:"invoicesrpc"`

	// RouterRPC is a sub-RPC server the exposes functionality that allows
	// clients to send payments on the network, and perform Lightning
	// payment related queries such as requests for estimates of off-chain
	// fees.
	RouterRPC *routerrpc.Config `group:"routerrpc" namespace:"routerrpc"`

	// WatchtowerRPC is a sub-RPC server that exposes functionality allowing
	// clients to monitor and control their embedded watchtower.
	WatchtowerRPC *watchtowerrpc.Config `group:"watchtowerrpc" namespace:"watchtowerrpc"`

	// WatchtowerClientRPC is a sub-RPC server that exposes functionality
	// that allows clients to interact with the active watchtower client
	// instance within lnd in order to add, remove, list registered client
	// towers, etc.
	WatchtowerClientRPC *wtclientrpc.Config `group:"wtclientrpc" namespace:"wtclientrpc"`
}

// Config is the object which will hold all of the config parameters
type Config struct {
	DefaultDir            bool     `yaml:"DefaultDir" long:"defaultdir" description:"Whether Conduit writes files to default directory or not"`
	ConduitDir            string   `yaml:"ConduitDir" long:"conduitdir" description:"Path to conduit configuration file"`
	ConsoleOutput         bool     `yaml:"ConsoleOutput" long:"console-output" description:"Whether or not Conduit prints the log to the console"`
	ShowVersion           bool     `short:"v" long:"version" description:"Display version information and exit"`
	LndConfigPath         string   `short:"C" long:"configfile" description:"Path to configuration file"`
	LndShowVersion        bool     `short:"V" long:"lnd-version" description:"Display LND version information and exit"`
	LndDataDir            string   `short:"b" long:"datadir" description:"The directory to store lnd's data within"`
	LndSyncFreelist       bool     `long:"sync-freelist" description:"Whether the databases used within lnd should sync their freelist to disk. This is disabled by default resulting in improved memory performance during operation, but with an increase in startup time."`
	LndTLSCertPath        string   `long:"tlscertpath" description:"Path to write the TLS certificate for lnd's RPC and REST services"`
	LndTLSKeyPath         string   `long:"tlskeypath" description:"Path to write the TLS private key for lnd's RPC and REST services"`
	LndTLSExtraIPs        []string `long:"tlsextraip" description:"Adds an extra ip to the generated certificate"`
	LndTLSExtraDomains    []string `long:"tlsextradomain" description:"Adds an extra domain to the generated certificate"`
	LndTLSAutoRefresh     bool     `long:"tlsautorefresh" description:"Re-generate TLS certificate and key if the IPs or domains are changed"`
	LndTLSDisableAutofill bool     `long:"tlsdisableautofill" description:"Do not include the interface IPs or the system hostname in TLS certificate, use first --tlsextradomain as Common Name instead, if set"`
	LndTLSCertDuration    string   `long:"tlscertduration" description:"The duration for which the auto-generated TLS certificate will be valid for"`
	LndNoMacaroons        bool     `long:"no-macaroons" description:"Disable macaroon authentication, can only be used if server is not listening on a public interface."`
	LndAdminMacPath       string   `long:"adminmacaroonpath" description:"Path to write the admin macaroon for lnd's RPC and REST services if it doesn't exist"`
	LndReadMacPath        string   `long:"readonlymacaroonpath" description:"Path to write the read-only macaroon for lnd's RPC and REST services if it doesn't exist"`
	LndInvoiceMacPath     string   `long:"invoicemacaroonpath" description:"Path to the invoice-only macaroon for lnd's RPC and REST services if it doesn't exist"`
	LndLogDir             string   `long:"logdir" description:"Directory to log output."`
	LndMaxLogFiles        string   `long:"maxlogfiles" description:"Maximum logfiles to keep (0 for no rotation)"`
	LndMaxLogFileSize     string   `long:"maxlogfilesize" description:"Maximum logfile size in MB"`
	LndAcceptorTimeout    string   `long:"acceptortimeout" description:"Time after which an RPCAcceptor will time out and return false if it hasn't yet received a response"`
	LndLetsEncryptDir     string   `long:"letsencryptdir" description:"The directory to store Let's Encrypt certificates within"`
	LndLetsEncryptListen  string   `long:"letsencryptlisten" description:"The IP:port on which lnd will listen for Let's Encrypt challenges. Let's Encrypt will always try to contact on port 80. Often non-root processes are not allowed to bind to ports lower than 1024. This configuration option allows a different port to be used, but must be used in combination with port forwarding from port 80. This configuration can also be used to specify another IP address to listen on, for example an IPv6 address."`
	LndLetsEncryptDomain  string   `long:"letsencryptdomain" description:"Request a Let's Encrypt certificate for this domain. Note that the certicate is only requested and stored when the first rpc connection comes in."`
	LndRawRPCListeners    []string `long:"rpclisten" description:"Add an interface/port/socket to listen for RPC connections"`
	LndRawRESTListeners   []string `long:"restlisten" description:"Add an interface/port/socket to listen for REST connections"`
	LndRawListeners       []string `long:"listen" description:"Add an interface/port to listen for peer connections"`
	LndRawExternalIPs     []string `long:"externalip" description:"Add an ip:port to the list of local addresses we claim to listen on to peers. If a port is not specified, the default (9735) will be used regardless of other parameters"`
	LndExternalHosts      []string `long:"externalhosts" description:"A set of hosts that should be periodically resolved to announce IPs for"`
	LndRestCORS           []string `long:"restcors" description:"Add an ip:port/hostname to allow cross origin access from. To allow all origins, set as \"*\"."`
	LndDisableListen      bool     `long:"nolisten" description:"Disable listening for incoming peer connections"`
	LndDisableRest        bool     `long:"norest" description:"Disable REST API"`
	LndDisableRestTLS     bool     `long:"no-rest-tls" description:"Disable TLS for REST connections"`
	LndWSPingInterval     string   `long:"ws-ping-interval" description:"The ping interval for REST based WebSocket connections, set to 0 to disable sending ping messages from the server side"`
	LndWSPongWait         string   `long:"ws-pong-wait" description:"The time we wait for a pong response message on REST based WebSocket connections before the connection is closed as inactive"`
	LndNAT                bool     `long:"nat" description:"Toggle NAT traversal support (using either UPnP or NAT-PMP) to automatically advertise your external IP address to the network -- NOTE this does not support devices behind multiple NATs"`
	LndMinBackoff         string   `long:"minbackoff" description:"Shortest backoff when reconnecting to persistent peers. Valid time units are {s, m, h}."`
	LndMaxBackoff         string   `long:"maxbackoff" description:"Longest backoff when reconnecting to persistent peers. Valid time units are {s, m, h}."`
	LndConnectionTimeout  string   `long:"connectiontimeout" description:"The timeout value for network connections. Valid time units are {ms, s, m, h}."`
	LndDebugLevel         string   `short:"d" long:"debuglevel" description:"Logging level for all subsystems {trace, debug, info, warn, error, critical} -- You may also specify <global-level>,<subsystem>=<level>,<subsystem2>=<level>,... to set the log level for individual subsystems -- Use show to list available subsystems"`
	LndCPUProfile         string   `long:"cpuprofile" description:"Write CPU profile to the specified file"`
	LndProfile            string   `long:"profile" description:"Enable HTTP profiling on either a port or host:port"`
	LndUnsafeDisconnect   bool     `long:"unsafe-disconnect" description:"DEPRECATED: Allows the rpcserver to intentionally disconnect from peers with open channels. THIS FLAG WILL BE REMOVED IN 0.10.0"`
	LndUnsafeReplay       bool     `long:"unsafe-replay" description:"Causes a link to replay the adds on its commitment txn after starting up, this enables testing of the sphinx replay logic."`
	LndMaxPendingChannels string   `long:"maxpendingchannels" description:"The maximum number of incoming pending channels permitted per peer."`
	LndBackupFilePath     string   `long:"backupfilepath" description:"The target location of the channel backup file"`
	LndFeeURL             string   `long:"feeurl" description:"Optional URL for external fee estimation. If no URL is specified, the method for fee estimation will depend on the chosen backend and network. Must be set for neutrino on mainnet."`

	LndBitcoinActive              bool     `long:"bitcoin.active" description:"If the chain should be active or not."`
	LndBitcoinChainDir            string   `long:"bitcoin.chaindir" description:"The directory to store the chain's data within."`
	LndBitcoinNode                string   `long:"bitcoin.node" description:"The blockchain interface to use." choice:"btcd" choice:"bitcoind" choice:"neutrino" choice:"ltcd" choice:"litecoind" choice:"nochainbackend"`
	LndBitcoinMainNet             bool     `long:"bitcoin.mainnet" description:"Use the main network"`
	LndBitcoinTestNet3            bool     `long:"bitcoin.testnet" description:"Use the test network"`
	LndBitcoinSimNet              bool     `long:"bitcoin.simnet" description:"Use the simulation test network"`
	LndBitcoinRegTest             bool     `long:"bitcoin.regtest" description:"Use the regression test network"`
	LndBitcoinSigNet              bool     `long:"bitcoin.signet" description:"Use the signet test network"`
	LndBitcoinSigNetChallenge     string   `long:"bitcoin.signetchallenge" description:"Connect to a custom signet network defined by this challenge instead of using the global default signet test network -- Can be specified multiple times"`
	LndBitcoinSigNetSeedNode      []string `long:"bitcoin.signetseednode" description:"Specify a seed node for the signet network instead of using the global default signet network seed nodes"`
	LndBitcoinDefaultNumChanConfs string   `long:"bitcoin.defaultchanconfs" description:"The default number of confirmations a channel must have before it's considered open. If this is not set, we will scale the value according to the channel size."`
	LndBitcoinDefaultRemoteDelay  string   `long:"bitcoin.defaultremotedelay" description:"The default number of blocks we will require our channel counterparty to wait before accessing its funds in case of unilateral close. If this is not set, we will scale the value according to the channel size."`
	LndBitcoinMaxLocalDelay       string   `long:"bitcoin.maxlocaldelay" description:"The maximum blocks we will allow our funds to be timelocked before accessing its funds in case of unilateral close. If a peer proposes a value greater than this, we will reject the channel."`
	LndBitcoinMinHTLCIn           string   `long:"bitcoin.minhtlc" description:"The smallest HTLC we are willing to accept on our channels, in millisatoshi"`
	LndBitcoinMinHTLCOut          string   `long:"bitcoin.minhtlcout" description:"The smallest HTLC we are willing to send out on our channels, in millisatoshi"`
	LndBitcoinBaseFee             string   `long:"bitcoin.basefee" description:"The base fee in millisatoshi we will charge for forwarding payments on our channels"`
	LndBitcoinFeeRate             string   `long:"bitcoin.feerate" description:"The fee rate used when forwarding payments on our channels. The total fee charged is basefee + (amount * feerate / 1000000), where amount is the forwarded amount."`
	LndBitcoinTimeLockDelta       string   `long:"bitcoin.timelockdelta" description:"The CLTV delta we will subtract from a forwarded HTLC's timelock value"`
	LndBitcoinDNSSeeds            []string `long:"bitcoin.dnsseed" description:"The seed DNS server(s) to use for initial peer discovery. Must be specified as a '<primary_dns>[,<soa_primary_dns>]' tuple where the SOA address is needed for DNS resolution through Tor but is optional for clearnet users. Multiple tuples can be specified, will overwrite the default seed servers."`

	LndBtcdDir        string `long:"btcd.dir" description:"The base directory that contains the node's data, logs, configuration file, etc."`
	LndBtcdRPCHost    string `long:"btcd.rpchost" description:"The daemon's rpc listening address. If a port is omitted, then the default port for the selected chain parameters will be used."`
	LndBtcdRPCUser    string `long:"btcd.rpcuser" description:"Username for RPC connections"`
	LndBtcdRPCPass    string `long:"btcd.rpcpass" default-mask:"-" description:"Password for RPC connections"`
	LndBtcdRPCCert    string `long:"btcd.rpccert" description:"File containing the daemon's certificate file"`
	LndBtcdRawRPCCert string `long:"btcd.rawrpccert" description:"The raw bytes of the daemon's PEM-encoded certificate chain which will be used to authenticate the RPC connection."`

	LndBitcoindDir                string `long:"bitcoind.dir" description:"The base directory that contains the node's data, logs, configuration file, etc."`
	LndBitcoindRPCHost            string `long:"bitcoind.rpchost" description:"The daemon's rpc listening address. If a port is omitted, then the default port for the selected chain parameters will be used."`
	LndBitcoindRPCUser            string `long:"bitcoind.rpcuser" description:"Username for RPC connections"`
	LndBitcoindRPCPass            string `long:"bitcoind.rpcpass" default-mask:"-" description:"Password for RPC connections"`
	LndBitcoindZMQPubRawBlock     string `long:"bitcoind.zmqpubrawblock" description:"The address listening for ZMQ connections to deliver raw block notifications"`
	LndBitcoindZMQPubRawTx        string `long:"bitcoind.zmqpubrawtx" description:"The address listening for ZMQ connections to deliver raw transaction notifications"`
	LndBitcoindEstimateMode       string `long:"bitcoind.estimatemode" description:"The fee estimate mode. Must be either ECONOMICAL or CONSERVATIVE."`
	LndBitcoindPrunedNodeMaxPeers string `long:"bitcoind.pruned-node-max-peers" description:"The maximum number of peers lnd will choose from the backend node to retrieve pruned blocks from. This only applies to pruned nodes."`

	LndNeutrinoAddPeers           []string `short:"a" long:"neutrino.addpeer" description:"Add a peer to connect with at startup"`
	LndNeutrinoConnectPeers       []string `long:"neutrino.connect" description:"Connect only to the specified peers at startup"`
	LndNeutrinoMaxPeers           string   `long:"neutrino.maxpeers" description:"Max number of inbound and outbound peers"`
	LndNeutrinoBanDuration        string   `long:"neutrino.banduration" description:"How long to ban misbehaving peers.  Valid time units are {s, m, h}.  Minimum 1 second"`
	LndNeutrinoBanThreshold       string   `long:"neutrino.banthreshold" description:"Maximum allowed ban score before disconnecting and banning misbehaving peers."`
	LndNeutrinoFeeURL             string   `long:"neutrino.feeurl" description:"DEPRECATED: Use top level 'feeurl' option. Optional URL for fee estimation. If a URL is not specified, static fees will be used for estimation."`
	LndNeutrinoAssertFilterHeader string   `long:"neutrino.assertfilterheader" description:"Optional filter header in height:hash format to assert the state of neutrino's filter header chain on startup. If the assertion does not hold, then the filter header chain will be re-synced from the genesis block."`
	LndNeutrinoUserAgentName      string   `long:"neutrino.useragentname" description:"Used to help identify ourselves to other bitcoin peers"`
	LndNeutrinoUserAgentVersion   string   `long:"neutrino.useragentversion" description:"Used to help identify ourselves to other bitcoin peers"`
	LndNeutrinoValidateChannels   bool     `long:"neutrino.validatechannels" description:"Validate every channel in the graph during sync by downloading the containing block. This is the inverse of routing.assumechanvalid, meaning that for Neutrino the validation is turned off by default for massively increased graph sync performance. This speedup comes at the risk of using an unvalidated view of the network for routing. Overwrites the value of routing.assumechanvalid if Neutrino is used. (default: false)"`
	LndNeutrinoBroadcastTimeout   string   `long:"neutrino.broadcasttimeout" description:"The amount of time to wait before giving up on a transaction broadcast attempt."`
	LndNeutrinoPersistFilters     bool     `long:"neutrino.persistfilters" description:"Whether compact filters fetched from the P2P network should be persisted to disk."`

	LndLitecoinActive              bool     `long:"litecoin.active" description:"If the chain should be active or not."`
	LndLitecoinChainDir            string   `long:"litecoin.chaindir" description:"The directory to store the chain's data within."`
	LndLitecoinNode                string   `long:"litecoin.node" description:"The blockchain interface to use." choice:"btcd" choice:"bitcoind" choice:"neutrino" choice:"ltcd" choice:"litecoind" choice:"nochainbackend"`
	LndLitecoinMainNet             bool     `long:"litecoin.mainnet" description:"Use the main network"`
	LndLitecoinTestNet3            bool     `long:"litecoin.testnet" description:"Use the test network"`
	LndLitecoinSimNet              bool     `long:"litecoin.simnet" description:"Use the simulation test network"`
	LndLitecoinRegTest             bool     `long:"litecoin.regtest" description:"Use the regression test network"`
	LndLitecoinSigNet              bool     `long:"litecoin.signet" description:"Use the signet test network"`
	LndLitecoinSigNetChallenge     string   `long:"litecoin.signetchallenge" description:"Connect to a custom signet network defined by this challenge instead of using the global default signet test network -- Can be specified multiple times"`
	LndLitecoinSigNetSeedNode      []string `long:"litecoin.signetseednode" description:"Specify a seed node for the signet network instead of using the global default signet network seed nodes"`
	LndLitecoinDefaultNumChanConfs string   `long:"litecoin.defaultchanconfs" description:"The default number of confirmations a channel must have before it's considered open. If this is not set, we will scale the value according to the channel size."`
	LndLitecoinDefaultRemoteDelay  string   `long:"litecoin.defaultremotedelay" description:"The default number of blocks we will require our channel counterparty to wait before accessing its funds in case of unilateral close. If this is not set, we will scale the value according to the channel size."`
	LndLitecoinMaxLocalDelay       string   `long:"litecoin.maxlocaldelay" description:"The maximum blocks we will allow our funds to be timelocked before accessing its funds in case of unilateral close. If a peer proposes a value greater than this, we will reject the channel."`
	LndLitecoinMinHTLCIn           string   `long:"litecoin.minhtlc" description:"The smallest HTLC we are willing to accept on our channels, in millisatoshi"`
	LndLitecoinMinHTLCOut          string   `long:"litecoin.minhtlcout" description:"The smallest HTLC we are willing to send out on our channels, in millisatoshi"`
	LndLitecoinBaseFee             string   `long:"litecoin.basefee" description:"The base fee in millisatoshi we will charge for forwarding payments on our channels"`
	LndLitecoinFeeRate             string   `long:"litecoin.feerate" description:"The fee rate used when forwarding payments on our channels. The total fee charged is basefee + (amount * feerate / 1000000), where amount is the forwarded amount."`
	LndLitecoinTimeLockDelta       string   `long:"litecoin.timelockdelta" description:"The CLTV delta we will subtract from a forwarded HTLC's timelock value"`
	LndLitecoinDNSSeeds            []string `long:"litecoin.dnsseed" description:"The seed DNS server(s) to use for initial peer discovery. Must be specified as a '<primary_dns>[,<soa_primary_dns>]' tuple where the SOA address is needed for DNS resolution through Tor but is optional for clearnet users. Multiple tuples can be specified, will overwrite the default seed servers."`

	LndLtcdDir        string `long:"ltcd.dir" description:"The base directory that contains the node's data, logs, configuration file, etc."`
	LndLtcdRPCHost    string `long:"ltcd.rpchost" description:"The daemon's rpc listening address. If a port is omitted, then the default port for the selected chain parameters will be used."`
	LndLtcdRPCUser    string `long:"ltcd.rpcuser" description:"Username for RPC connections"`
	LndLtcdRPCPass    string `long:"ltcd.rpcpass" default-mask:"-" description:"Password for RPC connections"`
	LndLtcdRPCCert    string `long:"ltcd.rpccert" description:"File containing the daemon's certificate file"`
	LndLtcdRawRPCCert string `long:"ltcd.rawrpccert" description:"The raw bytes of the daemon's PEM-encoded certificate chain which will be used to authenticate the RPC connection."`

	LndLitecoindDir                string `long:"litecoind.dir" description:"The base directory that contains the node's data, logs, configuration file, etc."`
	LndLitecoindRPCHost            string `long:"litecoind.rpchost" description:"The daemon's rpc listening address. If a port is omitted, then the default port for the selected chain parameters will be used."`
	LndLitecoindRPCUser            string `long:"litecoind.rpcuser" description:"Username for RPC connections"`
	LndLitecoindRPCPass            string `long:"litecoind.rpcpass" default-mask:"-" description:"Password for RPC connections"`
	LndLitecoindZMQPubRawBlock     string `long:"litecoind.zmqpubrawblock" description:"The address listening for ZMQ connections to deliver raw block notifications"`
	LndLitecoindZMQPubRawTx        string `long:"litecoind.zmqpubrawtx" description:"The address listening for ZMQ connections to deliver raw transaction notifications"`
	LndLitecoindEstimateMode       string `long:"litecoind.estimatemode" description:"The fee estimate mode. Must be either ECONOMICAL or CONSERVATIVE."`
	LndLitecoindPrunedNodeMaxPeers string `long:"litecoind.pruned-node-max-peers" description:"The maximum number of peers lnd will choose from the backend node to retrieve pruned blocks from. This only applies to pruned nodes."`

	LndBlockCacheSize string `long:"blockcachesize" description:"The maximum capacity of the block cache"`

	LndAutopilotActive         string            `long:"autopilot.active" description:"If the autopilot agent should be active or not."`
	LndAutopilotHeuristic      map[string]string `long:"autopilot.heuristic" description:"Heuristic to activate, and the weight to give it during scoring."`
	LndAutopilotMaxChannels    string            `long:"autopilot.maxchannels" description:"The maximum number of channels that should be created"`
	LndAutopilotAllocation     string            `long:"autopilot.allocation" description:"The percentage of total funds that should be committed to automatic channel establishment"`
	LndAutopilotMinChannelSize string            `long:"autopilot.minchansize" description:"The smallest channel that the autopilot agent should create"`
	LndAutopilotMaxChannelSize string            `long:"autopilot.maxchansize" description:"The largest channel that the autopilot agent should create"`
	LndAutopilotPrivate        bool              `long:"autopilot.private" description:"Whether the channels created by the autopilot agent should be private or not. Private channels won't be announced to the network."`
	LndAutopilotMinConfs       string            `long:"autopilot.minconfs" description:"The minimum number of confirmations each of your inputs in funding transactions created by the autopilot agent must have."`
	LndAutopilotConfTarget     string            `long:"autopilot.conftarget" description:"The confirmation target (in blocks) for channels opened by autopilot."`

	LndTorActive                      bool   `long:"tor.active" description:"Allow outbound and inbound connections to be routed through Tor"`
	LndTorSOCKS                       string `long:"tor.socks" description:"The host:port that Tor's exposed SOCKS5 proxy is listening on"`
	LndTorDNS                         string `long:"tor.dns" description:"The DNS server as host:port that Tor will use for SRV queries - NOTE must have TCP resolution enabled"`
	LndTorStreamIsolation             bool   `long:"tor.streamisolation" description:"Enable Tor stream isolation by randomizing user credentials for each connection."`
	LndTorSkipProxyForClearNetTargets bool   `long:"tor.skip-proxy-for-clearnet-targets" description:"Allow the node to establish direct connections to services not running behind Tor."`
	LndTorControl                     string `long:"tor.control" description:"The host:port that Tor is listening on for Tor control connections"`
	LndTorTargetIPAddress             string `long:"tor.targetipaddress" description:"IP address that Tor should use as the target of the hidden service"`
	LndTorPassword                    string `long:"tor.password" description:"The password used to arrive at the HashedControlPassword for the control port. If provided, the HASHEDPASSWORD authentication method will be used instead of the SAFECOOKIE one."`
	LndTorV2                          bool   `long:"tor.v2" description:"Automatically set up a v2 onion service to listen for inbound connections"`
	LndTorV3                          bool   `long:"tor.v3" description:"Automatically set up a v3 onion service to listen for inbound connections"`
	LndTorPrivateKeyPath              string `long:"tor.privatekeypath" description:"The path to the private key of the onion service being created"`
	LndTorWatchtowerKeyPath           string `long:"tor.watchtowerkeypath" description:"The path to the private key of the watchtower onion service being created"`

	LndSignRPCSignerMacPath         string `long:"signrpc.signermacaroonpath" description:"Path to the signer macaroon"`
	LndWalletRPCWalletKitMacPath    string `long:"walletrpc.walletkitmacaroonpath" description:"Path to the wallet kit macaroon"`
	LndChainRPCChainNotifierMacPath string `long:"chainrpc.notifiermacaroonpath" description:"Path to the chain notifier macaroon"`
	LndRouterRPCRouterMacPath       string `long:"routerrpc.routermacaroonpath" description:"Path to the router macaroon"`
	LndRouterRPCMinRtProb           string `long:"routerrpc.minrtprob" description:"Minimum required route success probability to attempt the payment"`
	LndRouterRPCAPrioriHopProb      string `long:"routerrpc.apriorihopprob" description:"Assumed success probability of a hop in a route when no other information is available."`
	LndRouterRPCAPrioriWeight       string `long:"routerrpc.aprioriweight" description:"Weight of the a priori probability in success probability estimation. Valid values are in [0, 1]."`
	LndRouterRPCPenaltyHalfLife     string `long:"routerrpc.penaltyhalflife" description:"Defines the duration after which a penalized node or channel is back at 50% probability"`
	LndRouterRPCAttemptCost         string `long:"routerrpc.attemptcost" description:"The fixed (virtual) cost in sats of a failed payment attempt"`
	LndRouterRPCAttemptCostPPM      string `long:"routerrpc.attemptcostppm" description:"The proportional (virtual) cost in sats of a failed payment attempt expressed in parts per million of the total payment amount"`
	LndRouterRPCMaxMcHistory        string `long:"routerrpc.maxmchistory" description:"the maximum number of payment results that are held on disk by mission control"`
	LndRouterRPCMcFlushInterval     string `long:"routerrpc.mcflushinterval" description:"the timer interval to use to flush mission control state to the DB"`

	LndHodlExitSettle     bool `long:"hodl.exit-settle" description:"Instructs the node to drop ADDs for which it is the exit node, and to not settle back to the sender"`
	LndHodlAddIncoming    bool `long:"hodl.add-incoming" description:"Instructs the node to drop incoming ADDs before processing them in the incoming link"`
	LndHodlSettleIncoming bool `long:"hodl.settle-incoming" description:"Instructs the node to drop incoming SETTLEs before processing them in the incoming link"`
	LndHodlFailIncoming   bool `long:"hodl.fail-incoming" description:"Instructs the node to drop incoming FAILs before processing them in the incoming link"`
	LndHodlAddOutgoing    bool `long:"hodl.add-outgoing" description:"Instructs the node to drop outgoing ADDs before applying them to the channel state"`
	LndHodlSettleOutgoing bool `long:"hodl.settle-outgoing" description:"Instructs the node to drop outgoing SETTLEs before applying them to the channel state"`
	LndHodlFailOutgoing   bool `long:"hodl.fail-outgoing" description:"Instructs the node to drop outgoing FAILs before applying them to the channel state"`
	LndHodlCommit         bool `long:"hodl.commit" description:"Instructs the node to add HTLCs to its local commitment state and to open circuits for any ADDs, but abort before committing the changes"`
	LndHodlBogusSettle    bool `long:"hodl.bogus-settle" description:"Instructs the node to settle back any incoming HTLC with a bogus preimage"`

	LndNoNetBootstrap                bool   `long:"nobootstrap" description:"If true, then automatic network bootstrapping will not be attempted."`
	LndNoSeedBackup                  bool   `long:"noseedbackup" description:"If true, NO SEED WILL BE EXPOSED -- EVER, AND THE WALLET WILL BE ENCRYPTED USING THE DEFAULT PASSPHRASE. THIS FLAG IS ONLY FOR TESTING AND SHOULD NEVER BE USED ON MAINNET."`
	LndWalletUnlockPasswordFile      string `long:"wallet-unlock-password-file" description:"The full path to a file (or pipe/device) that contains the password for unlocking the wallet; if set, no unlocking through RPC is possible and lnd will exit if no wallet exists or the password is incorrect; if wallet-unlock-allow-create is also set then lnd will ignore this flag if no wallet exists and allow a wallet to be created through RPC."`
	LndWalletUnlockAllowCreate       bool   `long:"wallet-unlock-allow-create" description:"Don't fail with an error if wallet-unlock-password-file is set but no wallet exists yet."`
	LndResetWalletTransactions       bool   `long:"reset-wallet-transactions" description:"Removes all transaction history from the on-chain wallet on startup, forcing a full chain rescan starting at the wallet's birthday. Implements the same functionality as btcwallet's dropwtxmgr command. Should be set to false after successful execution to avoid rescanning on every restart of lnd."`
	LndCoinSelectionStrategy         string `long:"coin-selection-strategy" description:"The strategy to use for selecting coins for wallet transactions." choice:"largest" choice:"random"`
	LndPaymentsExpirationGracePeriod string `long:"payments-expiration-grace-period" description:"A period to wait before force closing channels with outgoing htlcs that have timed-out and are a result of this node initiated payments."`
	LndTrickleDelay                  string `long:"trickledelay" description:"Time in milliseconds between each release of announcements to the network"`
	LndChanEnableTimeout             string `long:"chan-enable-timeout" description:"The duration that a peer connection must be stable before attempting to send a channel update to reenable or cancel a pending disables of the peer's channels on the network."`
	LndChanDisableTimeout            string `long:"chan-disable-timeout" description:"The duration that must elapse after first detecting that an already active channel is actually inactive and sending channel update disabling it to the network. The pending disable can be canceled if the peer reconnects and becomes stable for chan-enable-timeout before the disable update is sent."`
	LndChanStatusSampleInterval      string `long:"chan-status-sample-interval" description:"The polling interval between attempts to detect if an active channel has become inactive due to its peer going offline."`
	LndHeightHintCacheQueryDisable   bool   `long:"height-hint-cache-query-disable" description:"Disable queries from the height-hint cache to try to recover channels stuck in the pending close state. Disabling height hint queries may cause longer chain rescans, resulting in a performance hit. Unset this after channels are unstuck so you can get better performance again."`
	LndAlias                         string `long:"alias" description:"The node alias. Used as a moniker by peers and intelligence services"`
	LndColor                         string `long:"color" description:"The color of the node in hex format (i.e. '#3399FF'). Used to customize node appearance in intelligence services"`
	LndMinChanSize                   string `long:"minchansize" description:"The smallest channel size (in satoshis) that we should accept. Incoming channels smaller than this will be rejected"`
	LndMaxChanSize                   string `long:"maxchansize" description:"The largest channel size (in satoshis) that we should accept. Incoming channels larger than this will be rejected"`
	LndCoopCloseTargetConfs          string `long:"coop-close-target-confs" description:"The target number of blocks that a cooperative channel close transaction should confirm in. This is used to estimate the fee to use as the lower bound during fee negotiation for the channel closure."`
	LndChannelCommitInterval         string `long:"channel-commit-interval" description:"The maximum time that is allowed to pass between receiving a channel state update and signing the next commitment. Setting this to a longer duration allows for more efficient channel operations at the cost of latency."`
	LndChannelCommitBatchSize        string `long:"channel-commit-batch-size" description:"The maximum number of channel state updates that is accumulated before signing a new commitment."`
	LndDefaultRemoteMaxHtlcs         string `long:"default-remote-max-htlcs" description:"The default max_htlc applied when opening or accepting channels. This value limits the number of concurrent HTLCs that the remote party can add to the commitment. The maximum possible value is 483."`
	LndNumGraphSyncPeers             string `long:"numgraphsyncpeers" description:"The number of peers that we should receive new graph updates from. This option can be tuned to save bandwidth for light clients or routing nodes."`
	LndHistoricalSyncInterval        string `long:"historicalsyncinterval" description:"The polling interval between historical graph sync attempts. Each historical graph sync attempt ensures we reconcile with the remote peer's graph from the genesis block."`
	LndIgnoreHistoricalGossipFilters bool   `long:"ignore-historical-gossip-filters" description:"If true, will not reply with historical data that matches the range specified by a remote peer's gossip_timestamp_filter. Doing so will result in lower memory and bandwidth requirements."`
	LndRejectPush                    bool   `long:"rejectpush" description:"If true, lnd will not accept channel opening requests with non-zero push amounts. This should prevent accidental pushes to merchant nodes."`
	LndRejectHTLC                    bool   `long:"rejecthtlc" description:"If true, lnd will not forward any HTLCs that are meant as onward payments. This option will still allow lnd to send HTLCs and receive HTLCs but lnd won't be used as a hop."`
	LndStaggerInitialReconnect       bool   `long:"stagger-initial-reconnect" description:"If true, will apply a randomized staggering between 0s and 30s when reconnecting to persistent peers on startup. The first 10 reconnections will be attempted instantly, regardless of the flag's value"`
	LndMaxOutgoingCltvExpiry         string `long:"max-cltv-expiry" description:"The maximum number of blocks funds could be locked up for when forwarding payments."`
	LndMaxChannelFeeAllocation       string `long:"max-channel-fee-allocation" description:"The maximum percentage of total funds that can be allocated to a channel's commitment fee. This only applies for the initiator of the channel. Valid values are within [0.1, 1]."`
	LndMaxCommitFeeRateAnchors       string `long:"max-commit-fee-rate-anchors" description:"The maximum fee rate in sat/vbyte that will be used for commitments of channels of the anchors type. Must be large enough to ensure transaction propagation"`
	LndDryRunMigration               bool   `long:"dry-run-migration" description:"If true, lnd will abort committing a migration if it would otherwise have been successful. This leaves the database unmodified, and still compatible with the previously active version of lnd."`
	LndEnableUpfrontShutdown         bool   `long:"enable-upfront-shutdown" description:"If true, option upfront shutdown script will be enabled. If peers that we open channels with support this feature, we will automatically set the script to which cooperative closes should be paid out to on channel open. This offers the partial protection of a channel peer disconnecting from us if cooperative close is attempted with a different script."`
	LndAcceptKeySend                 bool   `long:"accept-keysend" description:"If true, spontaneous payments through keysend will be accepted. [experimental]"`
	LndAcceptAMP                     bool   `long:"accept-amp" description:"If true, spontaneous payments via AMP will be accepted."`
	LndKeysendHoldTime               string `long:"keysend-hold-time" description:"If non-zero, keysend payments are accepted but not immediately settled. If the payment isn't settled manually after the specified time, it is canceled automatically. [experimental]"`
	LndGcCanceledInvoicesOnStartup   bool   `long:"gc-canceled-invoices-on-startup" description:"If true, we'll attempt to garbage collect canceled invoices upon start."`
	LndGcCanceledInvoicesOnTheFly    bool   `long:"gc-canceled-invoices-on-the-fly" description:"If true, we'll delete newly canceled invoices on the fly."`
	LndDustThreshold                 string `long:"dust-threshold" description:"Sets the dust sum threshold in satoshis for a channel after which dust HTLC's will be failed."`

	LndInvoicesHoldExpiryDelta string `long:"invoices.holdexpirydelta" description:"The number of blocks before a hold invoice's htlc expires that the invoice should be canceled to prevent a force close. Force closes will not be prevented if this value is not greater than DefaultIncomingBroadcastDelta."`

	LndRoutingAssumeChannelValid  bool `long:"routing.assumechanvalid" description:"Skip checking channel spentness during graph validation. This speedup comes at the risk of using an unvalidated view of the network for routing. (default: false)"`
	LndRoutingStrictZombiePruning bool `long:"routing.strictgraphpruning" description:"If true, then the graph will be pruned more aggressively for zombies. In practice this means that edges with a single stale edge will be considered a zombie."`

	LndGossipPinnedSyncersRaw      []string `long:"gossip.pinned-syncers" description:"A set of peers that should always remain in an active sync state, which can be used to closely synchronize the routing tables of two nodes. The value should be comma separated list of hex-encoded pubkeys. Connected peers matching this pubkey will remain active for the duration of the connection and not count towards the NumActiveSyncer count."`
	LndGossipMaxChannelUpdateBurst string   `long:"gossip.max-channel-update-burst" description:"The maximum number of updates for a specific channel and direction that lnd will accept over the channel update interval."`
	LndGossipChannelUpdateInterval string   `long:"gossip.channel-update-interval" description:"The interval used to determine how often lnd should allow a burst of new updates for a specific channel and direction."`

	LndWorkersRead  string `long:"workers.read" description:"Maximum number of concurrent read pool workers. This number should be proportional to the number of peers."`
	LndWorkersWrite string `long:"workers.write" description:"Maximum number of concurrent write pool workers. This number should be proportional to the number of CPUs on the host. "`
	LndWorkersSig   string `long:"workers.sig" description:"Maximum number of concurrent sig pool workers. This number should be proportional to the number of CPUs on the host."`

	LndCachesRejectCacheSize       string `long:"caches.reject-cache-size" description:"Maximum number of entries contained in the reject cache, which is used to speed up filtering of new channel announcements and channel updates from peers. Each entry requires 25 bytes."`
	LndCachesChannelCacheSize      string `long:"caches.channel-cache-size" description:"Maximum number of entries contained in the channel cache, which is used to reduce memory allocations from gossip queries from peers. Each entry requires roughly 2Kb."`
	LndCachesRPCGraphCacheDuration string `long:"caches.rpc-graph-cache-duration" description:"The period of time expressed as a duration (1s, 1m, 1h, etc) that the RPC response to DescribeGraph should be cached for."`

	LndPrometheusListen string `long:"prometheus.listen" description:"the interface we should listen on for Prometheus"`
	LndPrometheusEnable bool   `long:"prometheus.enable" description:"enable Prometheus exporting of lnd gRPC performance metrics."`

	LndWtClientActive           bool     `long:"wtclient.active" description:"Whether the daemon should use private watchtowers to back up revoked channel states."`
	LndWtClientPrivateTowerURIs []string `long:"wtclient.private-tower-uris" description:"(Deprecated) Specifies the URIs of private watchtowers to use in backing up revoked states. URIs must be of the form <pubkey>@<addr>. Only 1 URI is supported at this time, if none are provided the tower will not be enabled."`
	LndWtClientSweepFeeRate     string   `long:"wtclient.sweep-fee-rate" description:"Specifies the fee rate in sat/byte to be used when constructing justice transactions sent to the watchtower."`

	LndWatchtowerActive         bool     `long:"watchtower.active" description:"If the watchtower should be active or not"`
	LndWatchtowerTowerDir       string   `long:"watchtower.towerdir" description:"Directory of the watchtower.db"`
	LndWatchtowerRawListeners   []string `long:"watchtower.listen" description:"Add interfaces/ports to listen for peer connections"`
	LndWatchtowerRawExternalIPs []string `long:"watchtower.externalip" description:"Add interfaces/ports where the watchtower can accept peer connections"`
	LndWatchtowerReadTimeout    string   `long:"watchtower.readtimeout" description:"Duration the watchtower server will wait for messages to be received before hanging up on clients"`
	LndWatchtowerWriteTimeout   string   `long:"watchtower.writetimeout" description:"Duration the watchtower server will wait for messages to be written before hanging up on client connections"`

	LndProtocolOptionsWumboChans            bool `long:"protocol.wumbo-channels" description:"if set, then lnd will create and accept requests for channels larger chan 0.16 BTC"`
	LndProtocolOptionsNoAnchors             bool `long:"protocol.no-anchors" description:"disable support for anchor commitments"`
	LndProtocolOptionsNoScriptEnforcedLease bool `long:"protocol.no-script-enforced-lease" description:"disable support for script enforced lease commitments"`

	LndLegacyOnionFormat     bool `long:"legacy.onion" description:"force node to not advertise the new modern TLV onion format"`
	LndLegacyCommitmentTweak bool `long:"legacy.committweak" description:"force node to not advertise the new commitment format"`

	LndAllowCircularRoute bool `long:"allow-circular-route" description:"If true, our node will allow htlc forwards that arrive and depart on the same channel."`

	LndChainBackendHealthInterval string `long:"healthcheck.chainbackend.interval" description:"How often to run a health check."`
	LndChainBackendHealthAttempts string `long:"healthcheck.chainbackend.attempts" description:"The number of calls we will make for the check before failing. Set this value to 0 to disable a check."`
	LndChainBackendHealthTimeout  string `long:"healthcheck.chainbackend.timeout" description:"The amount of time we allow the health check to take before failing due to timeout."`
	LndChainBackendHealthBackoff  string `long:"healthcheck.chainbackend.backoff" description:"The amount of time to back-off between failed health checks."`

	LndDiskHealthRequiredRemaining string `long:"healthcheck.diskspace.diskrequired" description:"The minimum ratio of free disk space to total capacity that we allow before shutting lnd down safely."`
	LndDiskHealthInterval          string `long:"healthcheck.diskspace.interval" description:"How often to run a health check."`
	LndDiskHealthAttempts          string `long:"healthcheck.diskspace.attempts" description:"The number of calls we will make for the check before failing. Set this value to 0 to disable a check."`
	LndDiskHealthTimeout           string `long:"healthcheck.diskspace.timeout" description:"The amount of time we allow the health check to take before failing due to timeout."`
	LndDiskHealthBackoff           string `long:"healthcheck.diskspace.backoff" description:"The amount of time to back-off between failed health checks."`

	LndTLSHealthInterval string `long:"healthcheck.tls.interval" description:"How often to run a health check."`
	LndTLSHealthAttempts string `long:"healthcheck.tls.attempts" description:"The number of calls we will make for the check before failing. Set this value to 0 to disable a check."`
	LndTLSHealthTimeout  string `long:"healthcheck.tls.timeout" description:"The amount of time we allow the health check to take before failing due to timeout."`
	LndTLSHealthBackoff  string `long:"healthcheck.tls.backoff" description:"The amount of time to back-off between failed health checks."`

	LndTorConnectionHealthInterval string `long:"healthcheck.torconnection.interval" description:"How often to run a health check."`
	LndTorConnectionHealthAttempts string `long:"healthcheck.torconnection.attempts" description:"The number of calls we will make for the check before failing. Set this value to 0 to disable a check."`
	LndTorConnectionHealthTimeout  string `long:"healthcheck.torconnection.timeout" description:"The amount of time we allow the health check to take before failing due to timeout."`
	LndTorConnectionHealthBackoff  string `long:"healthcheck.torconnection.backoff" description:"The amount of time to back-off between failed health checks."`

	LndRemoteSignerHealthInterval string `long:"healthcheck.remotesigner.interval" description:"How often to run a health check."`
	LndRemoteSignerHealthAttempts string `long:"healthcheck.remotesigner.attempts" description:"The number of calls we will make for the check before failing. Set this value to 0 to disable a check."`
	LndRemoteSignerHealthTimeout  string `long:"healthcheck.remotesigner.timeout" description:"The amount of time we allow the health check to take before failing due to timeout."`
	LndRemoteSignerHealthBackoff  string `long:"healthcheck.remotesigner.backoff" description:"The amount of time to back-off between failed health checks."`

	LndDBBackend             string `long:"db.backend" description:"The selected database backend."`
	LndDBBatchCommitInterval string `long:"db.batch-commit-interval" description:"The maximum duration the channel graph batch schedulers will wait before attempting to commit a batch of pending updates. This can be tradeoff database contenion for commit latency."`
	LndDBNoGraphCache        bool   `long:"db.no-graph-cache" description:"Don't use the in-memory graph cache for path finding. Much slower but uses less RAM. Can only be used with a bolt database backend."`

	LndEtcdEmbedded           bool   `long:"db.etcd.embedded" description:"Use embedded etcd instance instead of the external one. Note: use for testing only."`
	LndEtcdEmbeddedClientPort string `long:"db.etcd.embedded_client_port" description:"Client port to use for the embedded instance. Note: use for testing only."`
	LndEtcdEmbeddedPeerPort   string `long:"db.etcd.embedded_peer_port" description:"Peer port to use for the embedded instance. Note: use for testing only."`
	LndEtcdEmbeddedLogFile    string `long:"db.etcd.embedded_log_file" description:"Optional log file to use for embedded instance logs. note: use for testing only."`
	LndEtcdHost               string `long:"db.etcd.host" description:"Etcd database host."`
	LndEtcdUser               string `long:"db.etcd.user" description:"Etcd database user."`
	LndEtcdPass               string `long:"db.etcd.pass" description:"Password for the database user."`
	LndEtcdNamespace          string `long:"db.etcd.namespace" description:"The etcd namespace to use."`
	LndEtcdDisableTLS         bool   `long:"db.etcd.disabletls" description:"Disable TLS for etcd connection. Caution: use for development only."`
	LndEtcdCertFile           string `long:"db.etcd.cert_file" description:"Path to the TLS certificate for etcd RPC."`
	LndEtcdKeyFile            string `long:"db.etcd.key_file" description:"Path to the TLS private key for etcd RPC."`
	LndEtcdInsecureSkipVerify bool   `long:"db.etcd.insecure_skip_verify" description:"Whether we intend to skip TLS verification"`
	LndEtcdCollectStats       bool   `long:"db.etcd.collect_stats" description:"Whether to collect etcd commit stats."`
	LndEtcdMaxMsgSize         string `long:"db.etcd.max_msg_size" description:"The maximum message size in bytes that we may send to etcd."`

	LndBoltNoFreelistSync    bool   `long:"db.bolt.nofreelistsync" description:"Whether the databases used within lnd should sync their freelist to disk. This is set to true by default, meaning we don't sync the free-list resulting in imporved memory performance during operation, but with an increase in startup time."`
	LndBoltAutoCompact       bool   `long:"db.bolt.auto-compact" description:"Whether the databases used within lnd should automatically be compacted on every startup (and if the database has the configured minimum age). This is disabled by default because it requires additional disk space to be available during the compaction that is freed afterwards. In general compaction leads to smaller database files."`
	LndBoltAutoCompactMinAge string `long:"db.bolt.auto-compact-min-age" description:"How long ago the last compaction of a database file must be for it to be considered for auto compaction again. Can be set to 0 to compact on every startup."`
	LndBoltDBTimeout         string `long:"db.bolt.dbtimeout" description:"Specify the timeout value used when opening the database."`

	LndPostgresDsn            string `long:"db.postgres.dsn" description:"Database connection string."`
	LndPostgresTimeout        string `long:"db.postgres.timeout" description:"Database connection timeout. Set to zero to disable."`
	LndPostgresMaxConnections string `long:"db.postgres.maxconnections" description:"The maximum number of open connections to the database. Set to zero for unlimited."`

	LndClusterEnableLeaderElection bool   `long:"cluster.enable-leader-election" description:"Enables leader election if set."`
	LndClusterLeaderElector        string `long:"cluster.leader-elector" choice:"etcd" description:"Leader elector to use. Valid values: \"etcd\"."`
	LndClusterEtcdElectionPrefix   string `long:"cluster.etcd-election-prefix" description:"Election key prefix when using etcd leader elector. Defaults to \"/leader/\"."`
	LndClusterID                   string `long:"cluster.id" description:"Identifier for this node inside the cluster (used in leader election). Defaults to the hostname."`

	LndRPCMiddlewareEnable           bool     `long:"rpcmiddleware.enable" description:"Enable the RPC middleware interceptor functionality."`
	LndRPCMiddlewareInterceptTimeout string   `long:"rpcmiddleware.intercepttimeout" description:"Time after which a RPC middleware intercept request will time out and return an error if it hasn't yet received a response."`
	LndRPCMiddlewareMandatory        []string `long:"rpcmiddleware.addmandatory" description:"Add the named middleware to the list of mandatory middlewares. All RPC requests are blocked/denied if any of the mandatory middlewares is not registered. Can be specified multiple times."`

	Enable           bool   `long:"remotesigner.enable" description:"Use a remote signer for signing any on-chain related transactions or messages. Only recommended if local wallet is initialized as watch-only. Remote signer must use the same seed/root key as the local watch-only wallet but must have private keys."`
	RPCHost          string `long:"remotesigner.rpchost" description:"The remote signer's RPC host:port"`
	MacaroonPath     string `long:"remotesigner.macaroonpath" description:"The macaroon to use for authenticating with the remote signer"`
	TLSCertPath      string `long:"remotesigner.tlscertpath" description:"The TLS certificate to use for establishing the remote signer's identity"`
	Timeout          string `long:"remotesigner.timeout" description:"The timeout for connecting to and signing requests with the remote signer. Valid time units are {s, m, h}."`
	MigrateWatchOnly bool   `long:"remotesigner.migrate-wallet-to-watch-only" description:"If a wallet with private key material already exists, migrate it into a watch-only wallet on first startup. WARNING: This cannot be undone! Make sure you have backed up your seed before you use this flag! All private keys will be purged from the wallet after first unlock with this flag!"`
}

var (
	config_file_name string = "config.yaml"
	default_dir             = func() string {
		return utils.AppDataDir("conduit", false)
	}
	default_config = func() *Config {
		return &Config{
			DefaultDir:    true,
			ConduitDir:    default_dir(),
			ConsoleOutput: true,
		}
	}
)

// InitConfig returns the `Config` struct with either default values, values specified in `config.yaml` or command line flags
func InitConfig(isTesting bool) (*Config, error) {
	// Check if fmtd directory exists, if no then create it
	if !utils.FileExists(utils.AppDataDir("conduit", false)) {
		err := os.Mkdir(utils.AppDataDir("conduit", false), 0775)
		if err != nil {
			log.Println(err)
		}
	}
	config := &Config{}
	if utils.FileExists(path.Join(default_dir(), config_file_name)) {
		filename, _ := filepath.Abs(path.Join(default_dir(), config_file_name))
		config_file, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Println(err)
			return default_config(), nil
		}
		err = yaml.Unmarshal(config_file, config)
		if err != nil {
			log.Println(err)
			config = default_config()
		} else {
			// Need to check if any config parameters aren't defined in `config.yaml` and assign them a default value
			config = check_yaml_config(config)
		}
	} else {
		config = default_config()
	}
	if !isTesting {
		// now to parse the flags
		if _, err := flags.Parse(config); err != nil {
			return nil, err
		}
		if config.ShowVersion {
			fmt.Println(utils.AppName, "version", utils.AppVersion)
			os.Exit(0)
		}
	}
	return config, nil
}

// change_field changes the value of a specified field from the config struct
func change_field(field reflect.Value, new_value interface{}) {
	if field.IsValid() {
		if field.CanSet() {
			f := field.Kind()
			switch f {
			case reflect.String:
				if v, ok := new_value.(string); ok {
					field.SetString(v)
				} else {
					log.Fatal(fmt.Sprintf("Type of new_value: %v does not match the type of the field: string", new_value))
				}
			case reflect.Bool:
				if v, ok := new_value.(bool); ok {
					field.SetBool(v)
				} else {
					log.Fatal(fmt.Sprintf("Type of new_value: %v does not match the type of the field: bool", new_value))
				}
			case reflect.Int64:
				if v, ok := new_value.(int64); ok {
					field.SetInt(v)
				} else {
					log.Fatal(fmt.Sprintf("Type of new_value: %v does not match the type of the field: int64", new_value))
				}
			}
		}
	}
}

// check_yaml_config iterates over the Config struct fields and changes blank fields to default values
func check_yaml_config(config *Config) *Config {
	pv := reflect.ValueOf(config)
	v := pv.Elem()
	field_names := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		field_name := field_names.Field(i).Name
		switch field_name {
		case "ConduitDir":
			if f.String() == "" {
				change_field(f, default_dir())
				dld := v.FieldByName("DefaultDir")
				change_field(dld, true)
			}
		}
	}
	return config
}

// getInterfaceFromReflection returns an interface from a reflection
func getInterfaceFromReflection(fType reflect.Value) interface{} {
	if fType.IsValid() {
		return fType.Interface()
	}
	return nil
}

// GetConfigTagValues returns a map of the LND config parameters and there values as parsed from the command line
func (c *Config) GetConfigTagValues() []string {
	var tags []string
	pv := reflect.ValueOf(c)
	v := pv.Elem()
	field_names := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := field_names.Field(i)
		fType := v.Field(i)
		if strings.Contains(f.Name, "Lnd") {
			if !strings.Contains(f.Name, "ShowVersion") {
				if alias, ok := f.Tag.Lookup("long"); ok {
					inter := getInterfaceFromReflection(fType)
					switch q := inter.(type) {
					case string:
						if q != "" {
							tags = append(tags, fmt.Sprintf("--%v=%v", alias, q))
						}
					case []string:
						if len(q) != 0 {
							for _, arg := range q {
								tags = append(tags, fmt.Sprintf("--%v=%v", alias, arg))
							}
						}
					case bool:
						if q {
							tags = append(tags, fmt.Sprintf("--%v", alias))
						}
					case map[string]string:
						if len(q) != 0 {
							for jam, arg := range q {
								tags = append(tags, fmt.Sprintf("--%v=%v:%v", alias, jam, arg))
							}
						}
					}

				}
			}
		}
	}
	return tags
}
