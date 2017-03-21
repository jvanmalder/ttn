// Copyright © 2017 The Things Network
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package handler

import (
	"time"

	pb_broker "github.com/TheThingsNetwork/ttn/api/broker"
	"github.com/TheThingsNetwork/ttn/api/fields"
	"github.com/TheThingsNetwork/ttn/api/trace"
	"github.com/TheThingsNetwork/ttn/core/types"
)

// ResponseDeadline indicates how long
var ResponseDeadline = 100 * time.Millisecond

func (h *handler) HandleUplink(uplink *pb_broker.DeduplicatedUplinkMessage) (err error) {
	appID, devID := uplink.AppId, uplink.DevId
	
	dev, err := h.devices.Get(appID, devID)
	if err != nil {
		return err
	}
	
	location := dev.Location
//	ctx := h.Ctx.WithFields(log.Fields{
//		"AppID": appID,
//		"Location": location,
//		"DevID": devID,
//	})
	ctx := h.Ctx.WithFields(fields.Get(uplink))
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
	h.status.uplink.Mark(1)

	uplink.Trace = uplink.Trace.WithEvent(trace.ReceiveEvent)

	dev.StartUpdate()

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
	uplink.Trace = uplink.Trace.WithEvent("process uplink")

	// Run Uplink Processors
	for _, processor := range processors {
		err = processor(ctx, uplink, appUplink, dev)
		if err == ErrNotNeeded {
			err = nil
			return nil
		} else if err != nil {
			return err
		}
	}

	err = h.devices.Set(dev)
	if err != nil {
		return err
	}
	dev.StartUpdate()

	// Publish Uplink
	h.mqttUp <- appUplink
	if h.amqpEnabled {
		h.amqpUp <- appUplink
	}

	noDownlinkErrEvent := &types.DeviceEvent{
		AppID: appID,
		DevID: devID,
		Event: types.DownlinkErrorEvent,
		Data:  types.ErrorEventData{Error: "No gateways available for downlink"},
	}

	if dev.CurrentDownlink == nil {
		<-time.After(ResponseDeadline)

		queue, err := h.devices.DownlinkQueue(appID, devID)
		if err != nil {
			return err
		}

		if len, _ := queue.Length(); len > 0 {
			if uplink.ResponseTemplate != nil {
				next, err := queue.Next()
				if err != nil {
					return err
				}
				dev.CurrentDownlink = next
			} else {
				h.mqttEvent <- noDownlinkErrEvent
				return nil
			}
		}
	}

	if uplink.ResponseTemplate == nil {
		if dev.CurrentDownlink != nil {
			h.mqttEvent <- noDownlinkErrEvent
		}
		return nil
	}

	// Save changes (if any)
	err = h.devices.Set(dev)
	if err != nil {
		return err
	}

	// Prepare Downlink
	var appDownlink types.DownlinkMessage
	if dev.CurrentDownlink != nil {
		appDownlink = *dev.CurrentDownlink
	}
	appDownlink.AppID = uplink.AppId
	appDownlink.Location = location
	appDownlink.DevID = uplink.DevId
	downlink := uplink.ResponseTemplate
	downlink.Trace = uplink.Trace.WithEvent("prepare downlink")

	// Handle Downlink
	err = h.HandleDownlink(&appDownlink, downlink)
	if err != nil {
		return err
	}

	return nil
}
