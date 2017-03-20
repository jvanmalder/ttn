// Copyright © 2016 The Things Network
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package handler

import (
	"time"

	pb_broker "github.com/TheThingsNetwork/ttn/api/broker"
	"github.com/TheThingsNetwork/ttn/core/types"
	"github.com/apex/log"
)

// ResponseDeadline indicates how long
var ResponseDeadline = 100 * time.Millisecond

func (h *handler) HandleUplink(uplink *pb_broker.DeduplicatedUplinkMessage) (err error) {
	appID, devID := uplink.AppId, uplink.DevId
	// Find device for location
	dev, err := h.devices.Get(uplink.AppId, uplink.DevId)
	if err != nil {
		return err
	}
	location := dev.Location
	ctx := h.Ctx.WithFields(log.Fields{
		"AppID": appID,
		"Location": location,
		"DevID": devID,
	})
	start := time.Now()
	defer func() {
		if err != nil {
			h.mqttEvent <- &types.DeviceEvent{
				AppID: appID,
				Location: location,
				DevID: devID,
				Event: types.UplinkErrorEvent,
				Data:  types.ErrorEventData{Error: err.Error()},
			}
			ctx.WithError(err).Warn("Could not handle uplink")
		} else {
			ctx.WithField("Duration", time.Now().Sub(start)).Info("Handled uplink")
		}
	}()

	// Build AppUplink
	appUplink := &types.UplinkMessage{
		AppID: appID,
		Location: location,
		DevID: devID,
	}

	// Get Uplink Processors
	processors := []UplinkProcessor{
		h.ConvertFromLoRaWAN,
		h.ConvertMetadata,
		h.ConvertFieldsUp, // TODO: this uses the raw payload -> check if this should be removed or if it just succeeds
	}

	ctx.WithField("NumProcessors", len(processors)).Debug("Running Uplink Processors")

	// Run Uplink Processors
	for _, processor := range processors {
		err = processor(ctx, uplink, appUplink)
		if err == ErrNotNeeded {
			err = nil
			return nil
		} else if err != nil {
			return err
		}
	}

	// Publish Uplink
	h.mqttUp <- appUplink
	if h.amqpEnabled {
		h.amqpUp <- appUplink
	}

	<-time.After(ResponseDeadline)

	// Find scheduled downlink
	var appDownlink types.DownlinkMessage
	if dev.NextDownlink != nil {
		appDownlink = *dev.NextDownlink
	}

	if uplink.ResponseTemplate == nil {
		ctx.Debug("No Downlink Available")
		return nil
	}

	// Prepare Downlink
	downlink := uplink.ResponseTemplate
	appDownlink.AppID = uplink.AppId
	appDownlink.Location = location
	appDownlink.DevID = uplink.DevId

	// Handle Downlink
	err = h.HandleDownlink(&appDownlink, downlink)
	if err != nil {
		return err
	}

	// Clear Downlink
	dev.StartUpdate()
	dev.NextDownlink = nil
	err = h.devices.Set(dev)
	if err != nil {
		return err
	}

	return nil
}
