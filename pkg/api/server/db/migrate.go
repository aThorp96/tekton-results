// Copyright 2025 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package db defines database models for Result data.
package db

import (
	"fmt"
	"slices"
	"strings"

	"github.com/tektoncd/results/pkg/api/server/config"
	"gorm.io/gorm"
)

func PostAutoMigrate(db *gorm.DB, config *config.Config) error {
	if err := syncLabels(db, config.HIGH_TRAFFIC_LABELS); err != nil {
		return fmt.Errorf("error syncing HIGH_TRAFFIC_LABELS: %w", err)
	}
	return nil
}

func syncLabels(db *gorm.DB, releventLabels []string) error {
	if releventLabels == nil || len(releventLabels) == 0 {
		return nil
	}

	existingLabels := []string{}
	newLabels := []string{}
	oldLabels := []string{}

	rows, err := db.Find(&LabelKey{}).Rows()
	if err != nil {
		return fmt.Errorf("cannot get existing label keys: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var labelKey LabelKey
		if err := db.ScanRows(rows, &labelKey); err != nil {
			return fmt.Errorf("error scanning existing labels: %w", err)
		}
		existingLabels = append(existingLabels, labelKey.Key)
		if !slices.Contains(releventLabels, labelKey.Key) {
			oldLabels = append(oldLabels, labelKey.Key)
		}
	}

	labelKeys := make([]LabelKey, len(releventLabels))
	for _, label := range releventLabels {
		if !slices.Contains(existingLabels, label) {
			labelKeys = append(labelKeys, LabelKey{Key: label})
			newLabels = append(newLabels, label)
		}
	}

	if len(newLabels) > 0 {
		createResult := db.Create(&labelKeys)
		if createResult.Error != nil {
			return fmt.Errorf("error creating configured labels: %w", createResult.Error)
		}

		// TODO: reindex to create the record_labels association, since the has-lables relationship should be up-to-date
	}

	if len(oldLabels) > 0 {
		if deleteResult := db.Delete(&LabelKey{}, fmt.Sprintf("key NOT IN (%s)", strings.Join(releventLabels, ","))); deleteResult.Error != nil {
			return fmt.Errorf("error deleting old labels: %w", deleteResult.Error)
		}
	}

	return nil
}
