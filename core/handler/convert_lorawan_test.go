// Copyright © 2017 The Things Network
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package handler

import (
	"testing"
	pb_broker "github.com/TheThingsNetwork/ttn/api/broker"
	pb_protocol "github.com/TheThingsNetwork/ttn/api/protocol"
	pb_lorawan "github.com/TheThingsNetwork/ttn/api/protocol/lorawan"
	"github.com/TheThingsNetwork/ttn/core/component"
	"github.com/TheThingsNetwork/ttn/core/handler/device"
	"github.com/TheThingsNetwork/ttn/core/types"
	. "github.com/TheThingsNetwork/ttn/utils/testing"
	. "github.com/smartystreets/assertions"
)

func buildLorawanUplink(payload []byte) (*pb_broker.DeduplicatedUplinkMessage, *types.UplinkMessage) {
	ttnUp := &pb_broker.DeduplicatedUplinkMessage{
		DevId:   "devid",
		AppId:   "appid",
		Payload: payload,
		ProtocolMetadata: &pb_protocol.RxMetadata{Protocol: &pb_protocol.RxMetadata_Lorawan{
			Lorawan: &pb_lorawan.Metadata{
				FCnt: 1,
			},
		}},
	}
	appUp := &types.UplinkMessage{}
	return ttnUp, appUp
}

func TestConvertFromLoRaWAN(t *testing.T) {
	a := New(t)
	var wg WaitGroup
	h := &handler{
		Component: &component.Component{Ctx: GetLogger(t, "TestConvertFromLoRaWAN")},
		devices:   device.NewRedisDeviceStore(GetRedisClient(), "handler-test-convert-from-lorawan"),
		qEvent:    make(chan *types.DeviceEvent, 10),
	}
	device := &device.Device{
		DevID:           "devid",
		AppID:           "appid",
		CurrentDownlink: &types.DownlinkMessage{},
	}
	ttnUp, appUp := buildLorawanUplink([]byte{0x40, 0x04, 0x03, 0x02, 0x01, 0x20, 0x01, 0x00, 0x0A, 0x46, 0x55, 0x96, 0x42, 0x92, 0xF2})
	err := h.ConvertFromLoRaWAN(h.Ctx, ttnUp, appUp, device)
	a.So(err, ShouldBeNil)
	// fmt.Printf("TestConvertFromLoRaWAN ENCRYPTED Payload: %v\n", appUp.PayloadRaw)
	// a.So(appUp.PayloadRaw, ShouldResemble, []byte{0xaa, 0xbc}) // TODO: fix so that it resembles encrypted payload encoded in base64, not decrypted payload
	a.So(appUp.PayloadRaw, ShouldResemble, []byte{0x40, 0x04, 0x03, 0x02, 0x01, 0x20, 0x01, 0x00, 0x0A, 0x46, 0x55, 0x96, 0x42, 0x92, 0xF2})
	a.So(appUp.FCnt, ShouldEqual, 1)
	a.So(device.CurrentDownlink, ShouldBeNil)

	device.CurrentDownlink = &types.DownlinkMessage{Confirmed: true}

	ttnUp.UnmarshalPayload()
	ttnUp.Message.GetLorawan().MType = pb_lorawan.MType_CONFIRMED_UP
	ttnUp.Message.GetLorawan().GetMacPayload().FCnt++
	ttnUp.GetProtocolMetadata().GetLorawan().FCnt = ttnUp.Message.GetLorawan().GetMacPayload().FCnt
	ttnUp.Message.GetLorawan().GetMacPayload().Ack = false
	ttnUp.Message.GetLorawan().SetMIC(device.NwkSKey)
	ttnUp.Payload = ttnUp.Message.GetLorawan().PHYPayloadBytes()

	err = h.ConvertFromLoRaWAN(h.Ctx, ttnUp, appUp, device)
	a.So(err, ShouldBeNil)
	a.So(appUp.Confirmed, ShouldBeTrue)
	a.So(device.CurrentDownlink, ShouldNotBeNil)

	device.CurrentDownlink = &types.DownlinkMessage{Confirmed: true}

	wg.Add(1)
	go func() {
		<-h.qEvent
		wg.Done()
	}()

	ttnUp.UnmarshalPayload()
	ttnUp.Message.GetLorawan().MType = pb_lorawan.MType_CONFIRMED_UP
	ttnUp.Message.GetLorawan().GetMacPayload().FCnt++
	ttnUp.GetProtocolMetadata().GetLorawan().FCnt = ttnUp.Message.GetLorawan().GetMacPayload().FCnt
	ttnUp.Message.GetLorawan().GetMacPayload().Ack = true
	ttnUp.Message.GetLorawan().SetMIC(device.NwkSKey)
	ttnUp.Payload = ttnUp.Message.GetLorawan().PHYPayloadBytes()

	err = h.ConvertFromLoRaWAN(h.Ctx, ttnUp, appUp, device)
	a.So(err, ShouldBeNil)
	a.So(appUp.Confirmed, ShouldBeTrue)

	wg.Wait()
}

func buildLorawanDownlink(payload []byte) (*types.DownlinkMessage, *pb_broker.DownlinkMessage) {
	appDown := &types.DownlinkMessage{
		DevID:      "devid",
		AppID:      "appid",
		PayloadRaw: payload,
	}
	ttnDown := &pb_broker.DownlinkMessage{
		Payload: []byte{96, 4, 3, 2, 1, 0, 1, 0, 1, 0, 0, 0, 0},
		DownlinkOption: &pb_broker.DownlinkOption{
			ProtocolConfig: &pb_protocol.TxConfiguration{Protocol: &pb_protocol.TxConfiguration_Lorawan{
				Lorawan: &pb_lorawan.TxConfiguration{
					FCnt: 1,
				},
			}},
		},
	}
	return appDown, ttnDown
}

func TestConvertToLoRaWAN(t *testing.T) {
	a := New(t)
	h := &handler{
		Component: &component.Component{Ctx: GetLogger(t, "TestConvertToLoRaWAN")},
		devices:   device.NewRedisDeviceStore(GetRedisClient(), "handler-test-convert-to-lorawan"),
	}
	device := &device.Device{
		DevID: "devid",
		AppID: "appid",
	}
	appDown, ttnDown := buildLorawanDownlink([]byte{0xaa, 0xbc})
	err := h.ConvertToLoRaWAN(h.Ctx, appDown, ttnDown, device)
	a.So(err, ShouldBeNil)
	// a.So(ttnDown.Payload, ShouldResemble, []byte{0x60, 0x04, 0x03, 0x02, 0x01, 0x00, 0x01, 0x00, 0x01, 0xa1, 0x33, 0x68, 0x0A, 0x08, 0xBD})
	a.So(ttnDown.Payload, ShouldResemble, []byte{0xaa, 0xbc})

	appDown, ttnDown = buildLorawanDownlink([]byte{0xaa, 0xbc})
	appDown.FPort = 8
	err = h.ConvertToLoRaWAN(h.Ctx, appDown, ttnDown, device)
	a.So(err, ShouldBeNil)
	// a.So(ttnDown.Payload, ShouldResemble, []byte{0x60, 0x04, 0x03, 0x02, 0x01, 0x00, 0x01, 0x00, 0x08, 0xa1, 0x33, 0x41, 0xA9, 0xFA, 0x03})
	a.So(ttnDown.Payload, ShouldResemble, []byte{0xaa, 0xbc})
}
