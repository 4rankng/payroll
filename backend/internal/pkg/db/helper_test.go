package db

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapDatabaseErrorMapsPayrateTimelineConstraints(t *testing.T) {
	helper := NewDatabaseHelper(nil)

	t.Run("duplicate effective date", func(t *testing.T) {
		err := helper.WrapDatabaseError(errors.New("Error 1062: Duplicate entry '74-2026-08-22' for key 'uq_payrates_live_project_from_date'"))
		require.EqualError(t, err, "Không thể lưu mức lương: đã tồn tại cấu hình có cùng ngày hiệu lực")
	})

	t.Run("multiple open configurations", func(t *testing.T) {
		err := helper.WrapDatabaseError(errors.New("Error 1062: Duplicate entry '74' for key 'uq_payrates_one_open_per_project'"))
		require.EqualError(t, err, "Không thể lưu mức lương: dự án chỉ có thể có một cấu hình đang hiệu lực")
	})
}
