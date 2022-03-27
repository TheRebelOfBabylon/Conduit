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
	"time"

	"github.com/TheRebelOfBabylon/Conduit/utils"
	flags "github.com/jessevdk/go-flags"
	"github.com/lightningnetwork/lnd/htlcswitch/hodl"
	"github.com/lightningnetwork/lnd/lncfg"
	"github.com/lightningnetwork/lnd/lnrpc/autopilotrpc"
	"github.com/lightningnetwork/lnd/lnrpc/chainrpc"
	"github.com/lightningnetwork/lnd/lnrpc/invoicesrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/lightningnetwork/lnd/lnrpc/signrpc"
	"github.com/lightningnetwork/lnd/lnrpc/walletrpc"
	"github.com/lightningnetwork/lnd/lnrpc/watchtowerrpc"
	"github.com/lightningnetwork/lnd/lnrpc/wtclientrpc"
	yaml "gopkg.in/yaml.v2"
)

type subRPCServerConfigs struct {
	// SignRPC is a sub-RPC server that exposes signing of arbitrary inputs
	// as a gRPC service.
	SignRPC *signrpc.Config `group:"signrpc" namespace:"signrpc"`

	// WalletKitRPC is a sub-RPC server that exposes functionality allowing
	// a client to send transactions through a wallet, publish them, and
	// also requests keys and addresses under control of the backing
	// wallet.
	WalletKitRPC *walletrpc.Config `group:"walletrpc" namespace:"walletrpc"`

	// AutopilotRPC is a sub-RPC server that exposes methods on the running
	// autopilot as a gRPC service.
	AutopilotRPC *autopilotrpc.Config `group:"autopilotrpc" namespace:"autopilotrpc"`

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

type Chain struct {
	Active              bool     `long:"active" description:"If the chain should be active or not."`
	ChainDir            string   `long:"chaindir" description:"The directory to store the chain's data within."`
	Node                string   `long:"node" description:"The blockchain interface to use." choice:"btcd" choice:"bitcoind" choice:"neutrino" choice:"ltcd" choice:"litecoind" choice:"nochainbackend"`
	MainNet             bool     `long:"mainnet" description:"Use the main network"`
	TestNet3            bool     `long:"testnet" description:"Use the test network"`
	SimNet              bool     `long:"simnet" description:"Use the simulation test network"`
	RegTest             bool     `long:"regtest" description:"Use the regression test network"`
	SigNet              bool     `long:"signet" description:"Use the signet test network"`
	SigNetChallenge     string   `long:"signetchallenge" description:"Connect to a custom signet network defined by this challenge instead of using the global default signet test network -- Can be specified multiple times"`
	SigNetSeedNode      []string `long:"signetseednode" description:"Specify a seed node for the signet network instead of using the global default signet network seed nodes"`
	DefaultNumChanConfs int      `long:"defaultchanconfs" description:"The default number of confirmations a channel must have before it's considered open. If this is not set, we will scale the value according to the channel size."`
	DefaultRemoteDelay  int      `long:"defaultremotedelay" description:"The default number of blocks we will require our channel counterparty to wait before accessing its funds in case of unilateral close. If this is not set, we will scale the value according to the channel size."`
	MaxLocalDelay       uint16   `long:"maxlocaldelay" description:"The maximum blocks we will allow our funds to be timelocked before accessing its funds in case of unilateral close. If a peer proposes a value greater than this, we will reject the channel."`
	MinHTLCIn           uint64   `long:"minhtlc" description:"The smallest HTLC we are willing to accept on our channels, in millisatoshi"`
	MinHTLCOut          uint64   `long:"minhtlcout" description:"The smallest HTLC we are willing to send out on our channels, in millisatoshi"`
	BaseFee             uint64   `long:"basefee" description:"The base fee in millisatoshi we will charge for forwarding payments on our channels"`
	FeeRate             uint64   `long:"feerate" description:"The fee rate used when forwarding payments on our channels. The total fee charged is basefee + (amount * feerate / 1000000), where amount is the forwarded amount."`
	TimeLockDelta       uint32   `long:"timelockdelta" description:"The CLTV delta we will subtract from a forwarded HTLC's timelock value"`
	DNSSeeds            []string `long:"dnsseed" description:"The seed DNS server(s) to use for initial peer discovery. Must be specified as a '<primary_dns>[,<soa_primary_dns>]' tuple where the SOA address is needed for DNS resolution through Tor but is optional for clearnet users. Multiple tuples can be specified, will overwrite the default seed servers."`
}

// Config is the object which will hold all of the config parameters
type Config struct {
	DefaultDir                       bool                     `yaml:"DefaultDir" long:"defaultdir" description:"Whether Conduit writes files to default directory or not"`
	ConduitDir                       string                   `yaml:"ConduitDir" long:"conduitdir" description:"Path to conduit configuration file"`
	ConsoleOutput                    bool                     `yaml:"ConsoleOutput" long:"console-output" description:"Whether or not Conduit prints the log to the console"`
	ShowVersion                      bool                     `short:"v" long:"version" description:"Display version information and exit"`
	LndConfigPath                    string                   `short:"C" long:"configfile" description:"Path to configuration file"`
	LndShowVersion                   bool                     `short:"V" long:"lnd-version" description:"Display LND version information and exit"`
	LndDataDir                       string                   `short:"b" long:"datadir" description:"The directory to store lnd's data within"`
	LndSyncFreelist                  bool                     `long:"sync-freelist" description:"Whether the databases used within lnd should sync their freelist to disk. This is disabled by default resulting in improved memory performance during operation, but with an increase in startup time."`
	LndTLSCertPath                   string                   `long:"tlscertpath" description:"Path to write the TLS certificate for lnd's RPC and REST services"`
	LndTLSKeyPath                    string                   `long:"tlskeypath" description:"Path to write the TLS private key for lnd's RPC and REST services"`
	LndTLSExtraIPs                   []string                 `long:"tlsextraip" description:"Adds an extra ip to the generated certificate"`
	LndTLSExtraDomains               []string                 `long:"tlsextradomain" description:"Adds an extra domain to the generated certificate"`
	LndTLSAutoRefresh                bool                     `long:"tlsautorefresh" description:"Re-generate TLS certificate and key if the IPs or domains are changed"`
	LndTLSDisableAutofill            bool                     `long:"tlsdisableautofill" description:"Do not include the interface IPs or the system hostname in TLS certificate, use first --tlsextradomain as Common Name instead, if set"`
	LndTLSCertDuration               time.Duration            `long:"tlscertduration" description:"The duration for which the auto-generated TLS certificate will be valid for"`
	LndNoMacaroons                   bool                     `long:"no-macaroons" description:"Disable macaroon authentication, can only be used if server is not listening on a public interface."`
	LndAdminMacPath                  string                   `long:"adminmacaroonpath" description:"Path to write the admin macaroon for lnd's RPC and REST services if it doesn't exist"`
	LndReadMacPath                   string                   `long:"readonlymacaroonpath" description:"Path to write the read-only macaroon for lnd's RPC and REST services if it doesn't exist"`
	LndInvoiceMacPath                string                   `long:"invoicemacaroonpath" description:"Path to the invoice-only macaroon for lnd's RPC and REST services if it doesn't exist"`
	LndLogDir                        string                   `long:"logdir" description:"Directory to log output."`
	LndMaxLogFiles                   int                      `long:"maxlogfiles" description:"Maximum logfiles to keep (0 for no rotation)"`
	LndMaxLogFileSize                int                      `long:"maxlogfilesize" description:"Maximum logfile size in MB"`
	LndAcceptorTimeout               time.Duration            `long:"acceptortimeout" description:"Time after which an RPCAcceptor will time out and return false if it hasn't yet received a response"`
	LndLetsEncryptDir                string                   `long:"letsencryptdir" description:"The directory to store Let's Encrypt certificates within"`
	LndLetsEncryptListen             string                   `long:"letsencryptlisten" description:"The IP:port on which lnd will listen for Let's Encrypt challenges. Let's Encrypt will always try to contact on port 80. Often non-root processes are not allowed to bind to ports lower than 1024. This configuration option allows a different port to be used, but must be used in combination with port forwarding from port 80. This configuration can also be used to specify another IP address to listen on, for example an IPv6 address."`
	LndLetsEncryptDomain             string                   `long:"letsencryptdomain" description:"Request a Let's Encrypt certificate for this domain. Note that the certicate is only requested and stored when the first rpc connection comes in."`
	LndRawRPCListeners               []string                 `long:"rpclisten" description:"Add an interface/port/socket to listen for RPC connections"`
	LndRawRESTListeners              []string                 `long:"restlisten" description:"Add an interface/port/socket to listen for REST connections"`
	LndRawListeners                  []string                 `long:"listen" description:"Add an interface/port to listen for peer connections"`
	LndRawExternalIPs                []string                 `long:"externalip" description:"Add an ip:port to the list of local addresses we claim to listen on to peers. If a port is not specified, the default (9735) will be used regardless of other parameters"`
	LndExternalHosts                 []string                 `long:"externalhosts" description:"A set of hosts that should be periodically resolved to announce IPs for"`
	LndRestCORS                      []string                 `long:"restcors" description:"Add an ip:port/hostname to allow cross origin access from. To allow all origins, set as \"*\"."`
	LndDisableListen                 bool                     `long:"nolisten" description:"Disable listening for incoming peer connections"`
	LndDisableRest                   bool                     `long:"norest" description:"Disable REST API"`
	LndDisableRestTLS                bool                     `long:"no-rest-tls" description:"Disable TLS for REST connections"`
	LndWSPingInterval                time.Duration            `long:"ws-ping-interval" description:"The ping interval for REST based WebSocket connections, set to 0 to disable sending ping messages from the server side"`
	LndWSPongWait                    time.Duration            `long:"ws-pong-wait" description:"The time we wait for a pong response message on REST based WebSocket connections before the connection is closed as inactive"`
	LndNAT                           bool                     `long:"nat" description:"Toggle NAT traversal support (using either UPnP or NAT-PMP) to automatically advertise your external IP address to the network -- NOTE this does not support devices behind multiple NATs"`
	LndMinBackoff                    time.Duration            `long:"minbackoff" description:"Shortest backoff when reconnecting to persistent peers. Valid time units are {s, m, h}."`
	LndMaxBackoff                    time.Duration            `long:"maxbackoff" description:"Longest backoff when reconnecting to persistent peers. Valid time units are {s, m, h}."`
	LndConnectionTimeout             time.Duration            `long:"connectiontimeout" description:"The timeout value for network connections. Valid time units are {ms, s, m, h}."`
	LndDebugLevel                    string                   `short:"d" long:"debuglevel" description:"Logging level for all subsystems {trace, debug, info, warn, error, critical} -- You may also specify <global-level>,<subsystem>=<level>,<subsystem2>=<level>,... to set the log level for individual subsystems -- Use show to list available subsystems"`
	LndCPUProfile                    string                   `long:"cpuprofile" description:"Write CPU profile to the specified file"`
	LndProfile                       string                   `long:"profile" description:"Enable HTTP profiling on either a port or host:port"`
	LndUnsafeDisconnect              bool                     `long:"unsafe-disconnect" description:"DEPRECATED: Allows the rpcserver to intentionally disconnect from peers with open channels. THIS FLAG WILL BE REMOVED IN 0.10.0"`
	LndUnsafeReplay                  bool                     `long:"unsafe-replay" description:"Causes a link to replay the adds on its commitment txn after starting up, this enables testing of the sphinx replay logic."`
	LndMaxPendingChannels            int                      `long:"maxpendingchannels" description:"The maximum number of incoming pending channels permitted per peer."`
	LndBackupFilePath                string                   `long:"backupfilepath" description:"The target location of the channel backup file"`
	LndFeeURL                        string                   `long:"feeurl" description:"Optional URL for external fee estimation. If no URL is specified, the method for fee estimation will depend on the chosen backend and network. Must be set for neutrino on mainnet."`
	LndBitcoin                       *Chain                   `group:"Bitcoin" namespace:"bitcoin"`
	LndBtcdMode                      *lncfg.Btcd              `group:"btcd" namespace:"btcd"`
	LndBitcoindMode                  *lncfg.Bitcoind          `group:"bitcoind" namespace:"bitcoind"`
	LndNeutrinoMode                  *lncfg.Neutrino          `group:"neutrino" namespace:"neutrino"`
	LndLitecoin                      *Chain                   `group:"Litecoin" namespace:"litecoin"`
	LndLtcdMode                      *lncfg.Btcd              `group:"ltcd" namespace:"ltcd"`
	LndLitecoindMode                 *lncfg.Bitcoind          `group:"litecoind" namespace:"litecoind"`
	LndBlockCacheSize                uint64                   `long:"blockcachesize" description:"The maximum capacity of the block cache"`
	LndAutopilot                     *lncfg.AutoPilot         `group:"Autopilot" namespace:"autopilot"`
	LndTor                           *lncfg.Tor               `group:"Tor" namespace:"tor"`
	LndSubRPCServers                 *subRPCServerConfigs     `group:"subrpc"`
	LndHodl                          *hodl.Config             `group:"hodl" namespace:"hodl"`
	LndNoNetBootstrap                bool                     `long:"nobootstrap" description:"If true, then automatic network bootstrapping will not be attempted."`
	LndNoSeedBackup                  bool                     `long:"noseedbackup" description:"If true, NO SEED WILL BE EXPOSED -- EVER, AND THE WALLET WILL BE ENCRYPTED USING THE DEFAULT PASSPHRASE. THIS FLAG IS ONLY FOR TESTING AND SHOULD NEVER BE USED ON MAINNET."`
	LndWalletUnlockPasswordFile      string                   `long:"wallet-unlock-password-file" description:"The full path to a file (or pipe/device) that contains the password for unlocking the wallet; if set, no unlocking through RPC is possible and lnd will exit if no wallet exists or the password is incorrect; if wallet-unlock-allow-create is also set then lnd will ignore this flag if no wallet exists and allow a wallet to be created through RPC."`
	LndWalletUnlockAllowCreate       bool                     `long:"wallet-unlock-allow-create" description:"Don't fail with an error if wallet-unlock-password-file is set but no wallet exists yet."`
	LndResetWalletTransactions       bool                     `long:"reset-wallet-transactions" description:"Removes all transaction history from the on-chain wallet on startup, forcing a full chain rescan starting at the wallet's birthday. Implements the same functionality as btcwallet's dropwtxmgr command. Should be set to false after successful execution to avoid rescanning on every restart of lnd."`
	LndCoinSelectionStrategy         string                   `long:"coin-selection-strategy" description:"The strategy to use for selecting coins for wallet transactions." choice:"largest" choice:"random"`
	LndPaymentsExpirationGracePeriod time.Duration            `long:"payments-expiration-grace-period" description:"A period to wait before force closing channels with outgoing htlcs that have timed-out and are a result of this node initiated payments."`
	LndTrickleDelay                  int                      `long:"trickledelay" description:"Time in milliseconds between each release of announcements to the network"`
	LndChanEnableTimeout             time.Duration            `long:"chan-enable-timeout" description:"The duration that a peer connection must be stable before attempting to send a channel update to reenable or cancel a pending disables of the peer's channels on the network."`
	LndChanDisableTimeout            time.Duration            `long:"chan-disable-timeout" description:"The duration that must elapse after first detecting that an already active channel is actually inactive and sending channel update disabling it to the network. The pending disable can be canceled if the peer reconnects and becomes stable for chan-enable-timeout before the disable update is sent."`
	LndChanStatusSampleInterval      time.Duration            `long:"chan-status-sample-interval" description:"The polling interval between attempts to detect if an active channel has become inactive due to its peer going offline."`
	LndHeightHintCacheQueryDisable   bool                     `long:"height-hint-cache-query-disable" description:"Disable queries from the height-hint cache to try to recover channels stuck in the pending close state. Disabling height hint queries may cause longer chain rescans, resulting in a performance hit. Unset this after channels are unstuck so you can get better performance again."`
	LndAlias                         string                   `long:"alias" description:"The node alias. Used as a moniker by peers and intelligence services"`
	LndColor                         string                   `long:"color" description:"The color of the node in hex format (i.e. '#3399FF'). Used to customize node appearance in intelligence services"`
	LndMinChanSize                   int64                    `long:"minchansize" description:"The smallest channel size (in satoshis) that we should accept. Incoming channels smaller than this will be rejected"`
	LndMaxChanSize                   int64                    `long:"maxchansize" description:"The largest channel size (in satoshis) that we should accept. Incoming channels larger than this will be rejected"`
	LndCoopCloseTargetConfs          uint32                   `long:"coop-close-target-confs" description:"The target number of blocks that a cooperative channel close transaction should confirm in. This is used to estimate the fee to use as the lower bound during fee negotiation for the channel closure."`
	LndChannelCommitInterval         time.Duration            `long:"channel-commit-interval" description:"The maximum time that is allowed to pass between receiving a channel state update and signing the next commitment. Setting this to a longer duration allows for more efficient channel operations at the cost of latency."`
	LndChannelCommitBatchSize        uint32                   `long:"channel-commit-batch-size" description:"The maximum number of channel state updates that is accumulated before signing a new commitment."`
	LndDefaultRemoteMaxHtlcs         uint16                   `long:"default-remote-max-htlcs" description:"The default max_htlc applied when opening or accepting channels. This value limits the number of concurrent HTLCs that the remote party can add to the commitment. The maximum possible value is 483."`
	LndNumGraphSyncPeers             int                      `long:"numgraphsyncpeers" description:"The number of peers that we should receive new graph updates from. This option can be tuned to save bandwidth for light clients or routing nodes."`
	LndHistoricalSyncInterval        time.Duration            `long:"historicalsyncinterval" description:"The polling interval between historical graph sync attempts. Each historical graph sync attempt ensures we reconcile with the remote peer's graph from the genesis block."`
	LndIgnoreHistoricalGossipFilters bool                     `long:"ignore-historical-gossip-filters" description:"If true, will not reply with historical data that matches the range specified by a remote peer's gossip_timestamp_filter. Doing so will result in lower memory and bandwidth requirements."`
	LndRejectPush                    bool                     `long:"rejectpush" description:"If true, lnd will not accept channel opening requests with non-zero push amounts. This should prevent accidental pushes to merchant nodes."`
	LndRejectHTLC                    bool                     `long:"rejecthtlc" description:"If true, lnd will not forward any HTLCs that are meant as onward payments. This option will still allow lnd to send HTLCs and receive HTLCs but lnd won't be used as a hop."`
	LndStaggerInitialReconnect       bool                     `long:"stagger-initial-reconnect" description:"If true, will apply a randomized staggering between 0s and 30s when reconnecting to persistent peers on startup. The first 10 reconnections will be attempted instantly, regardless of the flag's value"`
	LndMaxOutgoingCltvExpiry         uint32                   `long:"max-cltv-expiry" description:"The maximum number of blocks funds could be locked up for when forwarding payments."`
	LndMaxChannelFeeAllocation       float64                  `long:"max-channel-fee-allocation" description:"The maximum percentage of total funds that can be allocated to a channel's commitment fee. This only applies for the initiator of the channel. Valid values are within [0.1, 1]."`
	LndMaxCommitFeeRateAnchors       uint64                   `long:"max-commit-fee-rate-anchors" description:"The maximum fee rate in sat/vbyte that will be used for commitments of channels of the anchors type. Must be large enough to ensure transaction propagation"`
	LndDryRunMigration               bool                     `long:"dry-run-migration" description:"If true, lnd will abort committing a migration if it would otherwise have been successful. This leaves the database unmodified, and still compatible with the previously active version of lnd."`
	LndEnableUpfrontShutdown         bool                     `long:"enable-upfront-shutdown" description:"If true, option upfront shutdown script will be enabled. If peers that we open channels with support this feature, we will automatically set the script to which cooperative closes should be paid out to on channel open. This offers the partial protection of a channel peer disconnecting from us if cooperative close is attempted with a different script."`
	LndAcceptKeySend                 bool                     `long:"accept-keysend" description:"If true, spontaneous payments through keysend will be accepted. [experimental]"`
	LndAcceptAMP                     bool                     `long:"accept-amp" description:"If true, spontaneous payments via AMP will be accepted."`
	LndKeysendHoldTime               time.Duration            `long:"keysend-hold-time" description:"If non-zero, keysend payments are accepted but not immediately settled. If the payment isn't settled manually after the specified time, it is canceled automatically. [experimental]"`
	LndGcCanceledInvoicesOnStartup   bool                     `long:"gc-canceled-invoices-on-startup" description:"If true, we'll attempt to garbage collect canceled invoices upon start."`
	LndGcCanceledInvoicesOnTheFly    bool                     `long:"gc-canceled-invoices-on-the-fly" description:"If true, we'll delete newly canceled invoices on the fly."`
	LndDustThreshold                 uint64                   `long:"dust-threshold" description:"Sets the dust sum threshold in satoshis for a channel after which dust HTLC's will be failed."`
	LndInvoices                      *lncfg.Invoices          `group:"invoices" namespace:"invoices"`
	LndRouting                       *lncfg.Routing           `group:"routing" namespace:"routing"`
	LndGossip                        *lncfg.Gossip            `group:"gossip" namespace:"gossip"`
	LndWorkers                       *lncfg.Workers           `group:"workers" namespace:"workers"`
	LndCaches                        *lncfg.Caches            `group:"caches" namespace:"caches"`
	LndPrometheus                    lncfg.Prometheus         `group:"prometheus" namespace:"prometheus"`
	LndWtClient                      *lncfg.WtClient          `group:"wtclient" namespace:"wtclient"`
	LndWatchtower                    *lncfg.Watchtower        `group:"watchtower" namespace:"watchtower"`
	LndProtocolOptions               *lncfg.ProtocolOptions   `group:"protocol" namespace:"protocol"`
	LndAllowCircularRoute            bool                     `long:"allow-circular-route" description:"If true, our node will allow htlc forwards that arrive and depart on the same channel."`
	LndHealthChecks                  *lncfg.HealthCheckConfig `group:"healthcheck" namespace:"healthcheck"`
	LndDB                            *lncfg.DB                `group:"db" namespace:"db"`
	LndCluster                       *lncfg.Cluster           `group:"cluster" namespace:"cluster"`
	LndRPCMiddleware                 *lncfg.RPCMiddleware     `group:"rpcmiddleware" namespace:"rpcmiddleware"`
	LndRemoteSigner                  *lncfg.RemoteSigner      `group:"remotesigner" namespace:"remotesigner"`
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
func InitConfig() (*Config, error) {
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
	// now to parse the flags
	if _, err := flags.Parse(config); err != nil {
		return nil, err
	}
	if config.ShowVersion {
		fmt.Println(utils.AppName, "version", utils.AppVersion)
		os.Exit(0)
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
