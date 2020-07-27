package awsapi

import (
	"reflect"
	"testing"
)

func TestStoreItems(t *testing.T) {
	startLocalDynamo(t)
	defer stopLocalDynamo()

	msg := &TItem{"foo", "bar"}
	_, err := testTable.StoreItem(msg)
	if err != nil {
		t.Error(err)
	}
	fmsg := &TItem{}
	err = testTable.FetchItem("foo", fmsg)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(*msg, *fmsg) {
		t.Errorf("%+v != %+v", fmsg, msg)
	}
}

func TestMessaging(t *testing.T) {
	startLocalDynamo(t)
	defer stopLocalDynamo()

	/*
		msg := &Msg{"foo", "bar"}
		err := testTable.StoreItem(msg)
		if
		msgs := &ListMsg{}
		err = msgs.FetchByUMS(testTable, "bar")
		if err != nil {
			t.Error(err)
		}
		if msgs.Len() != 1 {
			t.Errorf("Could not fetch messages for UMS bar")
		}

		if !reflect.DeepEqual(msg, msgs.At(0)) {
			t.Errorf("%+v != %+v", msgs.At(0), msg)
		}

		_, err = testTable.StoreItem(&Msg{"baz", "bar"})
		if err != nil {
			t.Error(err)
		}
		msgs = &ListMsg{}
		err = msgs.FetchByUMS(testTable, "baz")
		if err != nil {
			t.Error(err)
		}
		if msgs.Len() != 0 {
			t.Errorf("Expected 0 baz items, got: %v", msgs)
		}

		msgs = &ListMsg{}
		err = msgs.FetchByUMS(testTable, "bar")
		if err != nil {
			t.Error(err)
		}
		if msgs.Len() != 2 {
			t.Errorf("Expected 2 bar items, got: %v", msgs)
		}
	*/
}

/*
func TestDynamo(t *testing.T) {
t.Run("Messages", Messaging)
}*/
