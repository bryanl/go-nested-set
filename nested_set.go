package nested_set

import (
	"github.com/thoas/go-funk"
	"gorm.io/gorm"
)

type MoveDirection int

const (
	MoveDirectionLeft  MoveDirection = 1
	MoveDirectionRight MoveDirection = 2
	MoveDirectionInner MoveDirection = 3
)

func MoveTo(db *gorm.DB, target Category, to Category, direction MoveDirection) error {
	var right, depthChange int
	var newParentId int64
	if direction == MoveDirectionLeft || direction == MoveDirectionRight {
		newParentId = to.ParentId
		depthChange = to.Depth - target.Depth
		right = to.Rgt
		if direction == MoveDirectionLeft {
			right = to.Lft - 1
		}
	} else {
		newParentId = to.ID
		depthChange = to.Depth + 1 - target.Depth
		right = to.Lft
	}
	moveToRightOfPosition(db, target, right, depthChange, newParentId)
	return nil
}

func moveToRightOfPosition(db *gorm.DB, target Category, position, depthChange int, newParentId int64) (err error) {
	targetRight := target.Rgt
	targetLeft := target.Lft
	targetWidth := targetRight - targetLeft + 1

	targets, err := findCategories(db, targetLeft, targetRight)
	if err != nil {
		return
	}

	targetIds := funk.Map(targets, func(c Category) int64 {
		return c.ID
	}).([]int64)

	var moveStep, affectedStep, affectedGte, affectedLte int
	moveStep = position - targetLeft + 1
	if moveStep < 0 {
		affectedGte = position + 1
		affectedLte = targetLeft - 1
		affectedStep = targetWidth
	} else {
		affectedGte = targetRight + 1
		affectedLte = position
		affectedStep = targetWidth * -1
		// 向后移需要减去本身的宽度
		moveStep = moveStep - targetWidth
	}

	err = moveAffected(db, affectedGte, affectedLte, affectedStep)
	if err != nil {
		return
	}

	err = moveTarget(db, target.ID, targetIds, moveStep, depthChange, newParentId)
	if err != nil {
		return
	}

	return
}

func moveTarget(db *gorm.DB, targetId int64, targetIds []int64, step, depthChange int, newParentId int64) (err error) {
	sql := `
UPDATE categories
SET lft=lft+?,
	rgt=rgt+?,
	depth=depth+?
WHERE id IN (?);
  `
	err = db.Exec(sql, step, step, depthChange, targetIds).Error
	if err != nil {
		return
	}
	return db.Exec("UPDATE categories SET parent_id=? WHERE id=?", newParentId, targetId).Error
}

func moveAffected(db *gorm.DB, gte, lte, step int) (err error) {
	sql := `
UPDATE categories
SET lft=(CASE WHEN lft>=? THEN lft+? ELSE lft END),
	rgt=(CASE WHEN rgt<=? THEN rgt+? ELSE rgt END)
WHERE (lft BETWEEN ? AND ?) OR (rgt BETWEEN ? AND ?);
  `
	return db.Debug().Exec(sql, gte, step, lte, step, gte, lte, gte, lte).Error
}

func findCategories(query *gorm.DB, left, right int) (categories []Category, err error) {
	err = query.Where("rgt>=? AND rgt <=?", left, right).Find(&categories).Error
	return
}

func findCategory(query *gorm.DB, id int64) (category Category, err error) {
	err = query.Where("id=?", id).Find(&category).Error
	return
}
