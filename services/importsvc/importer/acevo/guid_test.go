package acevo

import "testing"

func TestGUID_UUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		guid GUID
		want string
	}{
		{
			name: "driver guid from real file",
			guid: GUID{A: "5606161271971460949", B: "10414234914310523523"},
			want: "4DCD15F9-6169-3755-9086-CB7A42F30A83",
		},
		{
			name: "championship id from real file",
			guid: GUID{A: "13141904673075839097", B: "10429449666789929274"},
			want: "B6616CAB-DEBB-4079-90BC-D936DFD4713A",
		},
		{
			name: "season guid matches JSON season_guid field",
			// season_guid in the JSON is b6616cab-debb-4079-90bc-d936dfd4713a
			// championship_id encodes the same value
			guid: GUID{A: "13141904673075839097", B: "10429449666789929274"},
			want: "B6616CAB-DEBB-4079-90BC-D936DFD4713A",
		},
		{
			name: "invalid A returns empty string",
			guid: GUID{A: "not-a-number", B: "10414234914310523523"},
			want: "",
		},
		{
			name: "invalid B returns empty string",
			guid: GUID{A: "5606161271971460949", B: "not-a-number"},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.guid.UUID()
			if got != tc.want {
				t.Errorf("UUID() = %q, want %q", got, tc.want)
			}
		})
	}
}
