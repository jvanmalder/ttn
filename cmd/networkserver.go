// Copyright © 2016 The Things Network
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"gopkg.in/redis.v3"

	"github.com/TheThingsNetwork/ttn/core"
	"github.com/TheThingsNetwork/ttn/core/networkserver"
	"github.com/TheThingsNetwork/ttn/core/types"
	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// networkserverCmd represents the networkserver command
var networkserverCmd = &cobra.Command{
	Use:   "networkserver",
	Short: "The Things Network networkserver",
	Long:  ``,
	PreRun: func(cmd *cobra.Command, args []string) {
		ctx.WithFields(log.Fields{
			"server":   fmt.Sprintf("%s:%d", viper.GetString("networkserver.server-address"), viper.GetInt("networkserver.server-port")),
			"database": fmt.Sprintf("%s/%d", viper.GetString("networkserver.redis-address"), viper.GetInt("networkserver.redis-db")),
		}).Info("Using Configuration")
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx.Info("Starting")

		// Redis Client
		client := redis.NewClient(&redis.Options{
			Addr:     viper.GetString("networkserver.redis-address"),
			Password: "", // no password set
			DB:       int64(viper.GetInt("networkserver.redis-db")),
		})

		// Component
		component := core.NewComponent(ctx, "networkserver", fmt.Sprintf("%s:%d", viper.GetString("networkserver.server-address-announce"), viper.GetInt("networkserver.server-port")))

		// networkserver Server
		networkserver := networkserver.NewRedisNetworkServer(client, viper.GetInt("networkserver.net-id"))
		prefix, length, err := types.ParseDevAddrPrefix(viper.GetString("networkserver.prefix"))
		if err != nil {
			ctx.WithError(err).Fatal("Could not initialize networkserver")
		}
		err = networkserver.UsePrefix(prefix[:], length)
		if err != nil {
			ctx.WithError(err).Fatal("Could not initialize networkserver")
		}
		err = networkserver.Init(component)
		if err != nil {
			ctx.WithError(err).Fatal("Could not initialize networkserver")
		}

		// gRPC Server
		lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", viper.GetString("networkserver.server-address"), viper.GetInt("networkserver.server-port")))
		if err != nil {
			ctx.WithError(err).Fatal("Could not start gRPC server")
		}
		grpc := grpc.NewServer(component.ServerOptions()...)

		// Register and Listen
		networkserver.RegisterRPC(grpc)
		go grpc.Serve(lis)

		sigChan := make(chan os.Signal)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		ctx.WithField("signal", <-sigChan).Info("signal received")

		grpc.Stop()
	},
}

func init() {
	RootCmd.AddCommand(networkserverCmd)

	networkserverCmd.Flags().String("redis-address", "localhost:6379", "Redis server and port")
	viper.BindPFlag("networkserver.redis-address", networkserverCmd.Flags().Lookup("redis-address"))
	networkserverCmd.Flags().Int("redis-db", 0, "Redis database")
	viper.BindPFlag("networkserver.redis-db", networkserverCmd.Flags().Lookup("redis-db"))

	networkserverCmd.Flags().Int("net-id", 19, "LoRaWAN NetID")
	viper.BindPFlag("networkserver.net-id", networkserverCmd.Flags().Lookup("net-id"))

	networkserverCmd.Flags().String("prefix", "26000000/24", "LoRaWAN DevAddr Prefix that should be used for issuing device addresses")
	viper.BindPFlag("networkserver.prefix", networkserverCmd.Flags().Lookup("prefix"))

	networkserverCmd.Flags().String("server-address", "0.0.0.0", "The IP address to listen for communication")
	networkserverCmd.Flags().String("server-address-announce", "localhost", "The public IP address to announce")
	networkserverCmd.Flags().Int("server-port", 1903, "The port for communication")
	viper.BindPFlag("networkserver.server-address", networkserverCmd.Flags().Lookup("server-address"))
	viper.BindPFlag("networkserver.server-address-announce", networkserverCmd.Flags().Lookup("server-address-announce"))
	viper.BindPFlag("networkserver.server-port", networkserverCmd.Flags().Lookup("server-port"))
}