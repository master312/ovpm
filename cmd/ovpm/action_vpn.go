package main

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/asaskevich/govalidator"
	"github.com/master312/ovpm/api/pb"
	"github.com/master312/ovpm/errors"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
)

type vpnInitParams struct {
	rpcServURLStr    string
	hostname         string
	port             string
	proto            pb.VPNProto
	netCIDR          string
	dnsAddr          string
	keepalivePeriod  string
	keepaliveTimeout string
	useLZO           bool
}

func vpnStatusAction(rpcServURLStr string) error {
	// Parse RPC Server's URL.
	rpcSrvURL, err := url.Parse(rpcServURLStr)
	if err != nil {
		return errors.BadURL(rpcServURLStr, err)
	}

	// Create a gRPC connection to the server.
	rpcConn, err := grpcConnect(rpcSrvURL)
	if err != nil {
		exit(1)
		return err
	}
	defer rpcConn.Close()

	// Get services.
	var vpnSvc = pb.NewVPNServiceClient(rpcConn)

	// Request vpn status and user list from the services.
	vpnStatusResp, err := vpnSvc.Status(context.Background(), &pb.VPNStatusRequest{})
	if err != nil {
		err := errors.UnknownGRPCError(err)
		exit(1)
		return err
	}

	// Prepare table data and draw it on the terminal.
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"attribute", "value"})
	table.Append([]string{"Name", vpnStatusResp.Name})
	table.Append([]string{"Hostname", vpnStatusResp.Hostname})
	table.Append([]string{"Port", vpnStatusResp.Port})
	table.Append([]string{"Proto", vpnStatusResp.Proto})
	table.Append([]string{"Network", vpnStatusResp.Net})
	table.Append([]string{"Netmask", vpnStatusResp.Mask})
	table.Append([]string{"Created At", vpnStatusResp.CreatedAt})
	table.Append([]string{"DNS", vpnStatusResp.Dns})
	table.Append([]string{"Cert Exp", vpnStatusResp.ExpiresAt})
	table.Append([]string{"CA Cert Exp", vpnStatusResp.CaExpiresAt})
	table.Append([]string{"Use LZO", fmt.Sprintf("%t", vpnStatusResp.UseLzo)})

	table.Render()

	return nil
}

func vpnInitAction(params vpnInitParams) error {
	// Parse RPC Server's URL.
	rpcSrvURL, err := url.Parse(params.rpcServURLStr)
	if err != nil {
		return errors.BadURL(params.rpcServURLStr, err)
	}

	// Create a gRPC connection to the server.
	rpcConn, err := grpcConnect(rpcSrvURL)
	if err != nil {
		exit(1)
		return err
	}
	defer rpcConn.Close()

	// Prepare service caller..
	var vpnSvc = pb.NewVPNServiceClient(rpcConn)

	// Request init request from vpn service.
	_, err = vpnSvc.Init(context.Background(), &pb.VPNInitRequest{
		Hostname:         params.hostname,
		Port:             params.port,
		ProtoPref:        params.proto,
		IpBlock:          params.netCIDR,
		Dns:              params.dnsAddr,
		KeepalivePeriod:  params.keepalivePeriod,
		KeepaliveTimeout: params.keepaliveTimeout,
		UseLzo:           params.useLZO,
	})
	if err != nil {
		err := errors.UnknownGRPCError(err)
		exit(1)
		return err
	}

	logrus.WithFields(logrus.Fields{
		"SERVER":            "OpenVPN",
		"CIDR":              params.netCIDR,
		"PROTO":             params.proto,
		"HOSTNAME":          params.hostname,
		"PORT":              params.port,
		"KEEPALIVE_PERIOD":  params.keepalivePeriod,
		"KEEPALIVE_TIMEOUT": params.keepaliveTimeout,
		"USE_LZO":           params.useLZO,
	}).Infoln("vpn initialized")
	return nil
}

func vpnUpdateAction(rpcServURLStr string, netCIDR *string, dnsAddr *string, useLzo *bool) error {
	// Parse RPC Server's URL.
	rpcSrvURL, err := url.Parse(rpcServURLStr)
	if err != nil {
		return errors.BadURL(rpcServURLStr, err)
	}

	// Create a gRPC connection to the server.
	rpcConn, err := grpcConnect(rpcSrvURL)
	if err != nil {
		exit(1)
		return err
	}
	defer rpcConn.Close()

	// Set netCIDR if provided.
	var targetNetCIDR string
	if netCIDR != nil {
		if !govalidator.IsCIDR(*netCIDR) {
			return errors.NotCIDR(*netCIDR)
		}

		var response string
		for {
			fmt.Println("If you proceed, you will loose all your static ip definitions.")
			fmt.Println("Any user that is defined to have a static ip will be set to be dynamic again.")
			fmt.Println()
			fmt.Println("Are you sure ? (y/N)")
			_, err := fmt.Scanln(&response)
			if err != nil {
				logrus.Fatal(err)
				exit(1)
				return err
			}
			okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
			nokayResponses := []string{"n", "N", "no", "No", "NO"}
			if stringInSlice(response, okayResponses) {
				break
			} else if stringInSlice(response, nokayResponses) {
				return fmt.Errorf("user decided to cancel")
			}
		}
		targetNetCIDR = *netCIDR
	}

	// Set DNS address if provided.
	var targetDNSAddr string
	if dnsAddr != nil {
		if !govalidator.IsIPv4(*dnsAddr) {
			return errors.NotIPv4(*dnsAddr)
		}
		targetDNSAddr = *dnsAddr
	}

	// Set USE-LZO preference if provided.
	var targetLZOPref pb.VPNLZOPref
	if useLzo == nil {
		targetLZOPref = pb.VPNLZOPref_USE_LZO_NOPREF
	} else {
		if *useLzo == true {
			targetLZOPref = pb.VPNLZOPref_USE_LZO_ENABLE
		}
		if *useLzo == false {
			targetLZOPref = pb.VPNLZOPref_USE_LZO_DISABLE
		}
	}

	// Prepare service caller.
	var vpnSvc = pb.NewVPNServiceClient(rpcConn)

	// Request update request from vpn service.
	_, err = vpnSvc.Update(context.Background(), &pb.VPNUpdateRequest{
		IpBlock: targetNetCIDR,
		Dns:     targetDNSAddr,
		LzoPref: targetLZOPref,
	})
	if err != nil {
		err := errors.UnknownGRPCError(err)
		exit(1)
		return err
	}

	logrus.WithFields(logrus.Fields{
		"SERVER":  "OpenVPN",
		"CIDR":    targetNetCIDR,
		"DNS":     targetDNSAddr,
		"USE_LZO": targetLZOPref.String(),
	}).Infoln("changes applied")

	return nil
}

func vpnRestartAction(rpcServURLStr string) error {
	// Parse RPC Server's URL.
	rpcSrvURL, err := url.Parse(rpcServURLStr)
	if err != nil {
		return errors.BadURL(rpcServURLStr, err)
	}

	// Create a gRPC connection to the server.
	rpcConn, err := grpcConnect(rpcSrvURL)
	if err != nil {
		err := errors.UnknownGRPCError(err)
		exit(1)
		return err
	}
	defer rpcConn.Close()

	// Prepare service caller.
	var vpnSvc = pb.NewVPNServiceClient(rpcConn)

	_, err = vpnSvc.Restart(context.Background(), &pb.VPNRestartRequest{})
	if err != nil {
		err := errors.UnknownGRPCError(err)
		exit(1)
		return err
	}

	logrus.Info("ovpm server restarted")
	return nil
}
