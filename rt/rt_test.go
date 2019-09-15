package rt

import "testing"

func TestAddressQueueMap(t *testing.T) {

	cfg, err := loadConfig("rt-mail.test.json")
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}
	tests := [][]string{
		[]string{"example-support@rt.example", "support", "correspond"},
		[]string{"support-help-comment@rt.example", "support", "comment"},
		[]string{"help@rt.example", "help", "correspond"},
		[]string{"help@example.com", "example", "correspond"},
		[]string{"help-comment@example.com", "example", "comment"},
	}

	rt := RT{config: cfg}

	for _, test := range tests {
		queue, action := rt.addressToQueueAction(test[0])
		if queue != test[1] {
			t.Logf("testing '%s' got queue '%s' but expected '%s'", test[0], queue, test[1])
			t.Fail()
		}
		if action != test[2] {
			t.Logf("testing '%s' got action '%s' but expected '%s'", test[0], action, test[2])
			t.Fail()
		}

	}
}
