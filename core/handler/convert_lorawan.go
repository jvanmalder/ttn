// Copyright © 2017 The Things Network
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package handler

import (
	ttnlog "github.com/TheThingsNetwork/go-utils/log"
	pb_broker "github.com/TheThingsNetwork/ttn/api/broker"
	pb_lorawan "github.com/TheThingsNetwork/ttn/api/protocol/lorawan"
	"github.com/TheThingsNetwork/ttn/api/trace"
	"github.com/TheThingsNetwork/ttn/core/handler/device"
	"github.com/TheThingsNetwork/ttn/core/types"
	"github.com/TheThingsNetwork/ttn/utils/errors"
)

func (h *handler) ConvertFromLoRaWAN(ctx ttnlog.Interface, ttnUp *pb_broker.DeduplicatedUplinkMessage, appUp *types.UplinkMessage, dev *device.Device) (err error) {
	if err := ttnUp.UnmarshalPayload(); err != nil {
		return err
	}
	if ttnUp.GetMessage().GetLorawan() == nil {
		return errors.NewErrInvalidArgument("Uplink", "does not contain a LoRaWAN payload")
	}

	phyPayload := ttnUp.GetMessage().GetLorawan()
	macPayload := phyPayload.GetMacPayload()
	if macPayload == nil {
		return errors.NewErrInvalidArgument("Uplink", "does not contain a MAC payload")
	}

	ttnUp.Trace = ttnUp.Trace.WithEvent(trace.CheckMICEvent)
	err = phyPayload.ValidateMIC(dev.NwkSKey)
	if err != nil {
		return err
	}

	appUp.HardwareSerial = dev.DevEUI.String()

	appUp.FCnt = macPayload.FCnt
	if dev.FCntUp == appUp.FCnt {
		appUp.IsRetry = true
	}
	dev.FCntUp = appUp.FCnt
	
	if phyPayload.MType == pb_lorawan.MType_CONFIRMED_UP {
		appUp.Confirmed = true
	}

	// LoRaWAN: Decrypt
	if macPayload.FPort > 0 {
		appUp.PayloadRaw = ttnUp.Payload
		// Check if the above still runs ok, otherwise just put macPayload.FRMPayload or macPayload.FRMPayload[0]
		// in appUp.PayloadRaw
	}

	if dev.CurrentDownlink != nil && !appUp.IsRetry {
		// We have a downlink pending
		if dev.CurrentDownlink.Confirmed {
			// If it's confirmed, we can only unset it if we receive an ack.
			if macPayload.Ack {
				// Send event over MQTT
				h.qEvent <- &types.DeviceEvent{
					AppID: appUp.AppID,
					DevID: appUp.DevID,
					Event: types.DownlinkAckEvent,
					Data: types.DownlinkEventData{
						Message: dev.CurrentDownlink,
					},
				}
				dev.CurrentDownlink = nil
			}
		} else {
			// If it's unconfirmed, we can unset it.
			dev.CurrentDownlink = nil
		}
	}

	return nil
}

func (h *handler) ConvertToLoRaWAN(ctx ttnlog.Interface, appDown *types.DownlinkMessage, ttnDown *pb_broker.DownlinkMessage, dev *device.Device) error {
	// LoRaWAN: Unmarshal Downlink
//	var phyPayload lorawan.PHYPayload
//	err := phyPayload.UnmarshalBinary(ttnDown.Payload)
//	if err != nil {
//		return err
//	}
//	macPayload, ok := phyPayload.MACPayload.(*lorawan.MACPayload)
//	if !ok {
//		return errors.NewErrInvalidArgument("Downlink", "does not contain a MAC payload")
//	}
//	if ttnDown.DownlinkOption != nil && ttnDown.DownlinkOption.ProtocolConfig.GetLorawan() != nil {
//		macPayload.FHDR.FCnt = ttnDown.DownlinkOption.ProtocolConfig.GetLorawan().FCnt
//	}

	// Abort when downlink not needed
	if len(appDown.PayloadRaw) == 0 {// && !macPayload.FHDR.FCtrl.ACK && len(macPayload.FHDR.FOpts) == 0 {
		return ErrNotNeeded
	}

	// Set FPort
//	if appDown.FPort != 0 {
//		macPayload.FPort = &appDown.FPort
//	}

//	if appDown.Confirmed {
//		phyPayload.MHDR.MType = lorawan.ConfirmedDataDown
//	}
//
//	if queue, err := h.devices.DownlinkQueue(dev.AppID, dev.DevID); err == nil {
//		if length, _ := queue.Length(); length > 0 {
//			macPayload.FHDR.FCtrl.FPending = true
//		}
//	}

	// Set Payload
//	if len(appDown.PayloadRaw) > 0 {
//		ttnDown.Trace = ttnDown.Trace.WithEvent("set payload")
//		macPayload.FRMPayload = []lorawan.Payload{&lorawan.DataPayload{Bytes: appDown.PayloadRaw}}
//		if macPayload.FPort == nil || *macPayload.FPort == 0 {
//			macPayload.FPort = pointer.Uint8(1)
//		}
//	} else {
//		ttnDown.Trace = ttnDown.Trace.WithEvent("set empty payload")
//		macPayload.FRMPayload = []lorawan.Payload{}
//	}

	// Encrypt
//	err = phyPayload.EncryptFRMPayload(lorawan.AES128Key(dev.AppSKey))
//	if err != nil {
//		return err
//	}

	// Set MIC
//	err = phyPayload.SetMIC(lorawan.AES128Key(dev.NwkSKey))
//	if err != nil {
//		return err
//	}

	// Marshal
//	phyPayloadBytes, err := phyPayload.MarshalBinary()
//	if err != nil {
//		return err
//	}

	ttnDown.Payload = appDown.PayloadRaw

	return nil
}
