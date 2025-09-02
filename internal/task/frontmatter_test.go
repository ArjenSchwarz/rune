package task

import (
	"reflect"
	"testing"
)

func TestParseMetadataFlags(t *testing.T) {
	tests := map[string]struct {
		input   []string
		want    map[string]any
		wantErr bool
	}{
		"single_key_value": {
			input: []string{"key:value"},
			want: map[string]any{
				"key": "value",
			},
		},
		"multiple_key_values": {
			input: []string{"key1:value1", "key2:value2"},
			want: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
		},
		"multiple_values_same_key_creates_array": {
			input: []string{"tags:tag1", "tags:tag2", "tags:tag3"},
			want: map[string]any{
				"tags": []string{"tag1", "tag2", "tag3"},
			},
		},
		"mixed_single_and_array": {
			input: []string{"name:project", "tags:tag1", "version:1.0", "tags:tag2"},
			want: map[string]any{
				"name":    "project",
				"tags":    []string{"tag1", "tag2"},
				"version": "1.0",
			},
		},
		"colon_in_value": {
			input: []string{"url:https://example.com:8080", "time:10:30:45"},
			want: map[string]any{
				"url":  "https://example.com:8080",
				"time": "10:30:45",
			},
		},
		"empty_value": {
			input: []string{"key:"},
			want: map[string]any{
				"key": "",
			},
		},
		"empty_key_error": {
			input:   []string{":value"},
			wantErr: true,
		},
		"no_colon_error": {
			input:   []string{"invalid"},
			wantErr: true,
		},
		"empty_string_error": {
			input:   []string{""},
			wantErr: true,
		},
		"whitespace_preserved_in_values": {
			input: []string{"description:This is a test", "title:  Leading spaces"},
			want: map[string]any{
				"description": "This is a test",
				"title":       "  Leading spaces",
			},
		},
		"nested_key_not_supported": {
			input:   []string{"author.name:John Doe"},
			wantErr: true,
		},
		"dot_in_key_error": {
			input:   []string{"config.port:8080"},
			wantErr: true,
		},
		"invalid_key_characters": {
			input:   []string{"key-with-dash:value"},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParseMetadataFlags(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParseMetadataFlags() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr && !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseMetadataFlags() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMergeFrontMatter(t *testing.T) {
	tests := map[string]struct {
		existing *FrontMatter
		new      *FrontMatter
		want     *FrontMatter
		wantErr  bool
	}{
		"merge_empty_with_new": {
			existing: &FrontMatter{},
			new: &FrontMatter{
				References: []string{"doc1.md"},
				Metadata: map[string]any{
					"version": "1.0",
				},
			},
			want: &FrontMatter{
				References: []string{"doc1.md"},
				Metadata: map[string]any{
					"version": "1.0",
				},
			},
		},
		"append_references_without_deduplication": {
			existing: &FrontMatter{
				References: []string{"doc1.md", "doc2.md"},
			},
			new: &FrontMatter{
				References: []string{"doc2.md", "doc3.md"},
			},
			want: &FrontMatter{
				References: []string{"doc1.md", "doc2.md", "doc2.md", "doc3.md"},
				Metadata:   map[string]any{},
			},
		},
		"scalar_metadata_replacement": {
			existing: &FrontMatter{
				Metadata: map[string]any{
					"version": "1.0",
					"author":  "Alice",
				},
			},
			new: &FrontMatter{
				Metadata: map[string]any{
					"version": "2.0",
				},
			},
			want: &FrontMatter{
				References: []string{},
				Metadata: map[string]any{
					"version": "2.0",
					"author":  "Alice",
				},
			},
		},
		"array_metadata_appending": {
			existing: &FrontMatter{
				Metadata: map[string]any{
					"tags": []string{"tag1", "tag2"},
				},
			},
			new: &FrontMatter{
				Metadata: map[string]any{
					"tags": []string{"tag3"},
				},
			},
			want: &FrontMatter{
				References: []string{},
				Metadata: map[string]any{
					"tags": []string{"tag1", "tag2", "tag3"},
				},
			},
		},
		"type_conflict_error": {
			existing: &FrontMatter{
				Metadata: map[string]any{
					"value": "string",
				},
			},
			new: &FrontMatter{
				Metadata: map[string]any{
					"value": []string{"array"},
				},
			},
			wantErr: true,
		},
		"nested_map_not_supported": {
			existing: &FrontMatter{
				Metadata: map[string]any{
					"config": map[string]any{
						"host": "localhost",
					},
				},
			},
			new: &FrontMatter{
				Metadata: map[string]any{
					"config": map[string]any{
						"port": 9090,
					},
				},
			},
			wantErr: true,
		},
		"complex_merge_flat_only": {
			existing: &FrontMatter{
				References: []string{"ref1.md"},
				Metadata: map[string]any{
					"version": "1.0",
					"tags":    []string{"tag1"},
					"author":  "Alice",
				},
			},
			new: &FrontMatter{
				References: []string{"ref2.md"},
				Metadata: map[string]any{
					"version": "2.0",
					"tags":    []string{"tag2"},
					"email":   "alice@example.com",
				},
			},
			want: &FrontMatter{
				References: []string{"ref1.md", "ref2.md"},
				Metadata: map[string]any{
					"version": "2.0",
					"tags":    []string{"tag1", "tag2"},
					"author":  "Alice",
					"email":   "alice@example.com",
				},
			},
		},
		"nil_existing": {
			existing: nil,
			new: &FrontMatter{
				References: []string{"new.md"},
			},
			want: &FrontMatter{
				References: []string{"new.md"},
				Metadata:   map[string]any{},
			},
		},
		"nil_new": {
			existing: &FrontMatter{
				References: []string{"existing.md"},
			},
			new: nil,
			want: &FrontMatter{
				References: []string{"existing.md"},
				Metadata:   map[string]any{},
			},
		},
		"both_nil": {
			existing: nil,
			new:      nil,
			want: &FrontMatter{
				References: []string{},
				Metadata:   map[string]any{},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := MergeFrontMatter(tc.existing, tc.new)
			if (err != nil) != tc.wantErr {
				t.Errorf("MergeFrontMatter() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr && !reflect.DeepEqual(got, tc.want) {
				t.Errorf("MergeFrontMatter() = %v, want %v", got, tc.want)
			}
		})
	}
}
