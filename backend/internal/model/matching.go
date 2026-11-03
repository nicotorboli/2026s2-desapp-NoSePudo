package model

import "sort"

type Execution struct {
	RestingOrderID int64
	Quantity       int64
	Price          Credits
}

func MatchOrder(incoming Order, book []Order) []Execution {
	remaining := incoming.RemainingQuantity()
	if remaining <= 0 {
		return nil
	}

	eligible := eligibleRestingOrders(incoming, book)
	sortByPriceTimePriority(incoming.Side, eligible)

	executions := make([]Execution, 0, len(eligible))
	for i := range eligible {
		if remaining == 0 {
			break
		}

		quantity := min(remaining, eligible[i].RemainingQuantity())
		executions = append(executions, Execution{
			RestingOrderID: eligible[i].ID,
			Quantity:       quantity,
			Price:          eligible[i].Price,
		})
		remaining -= quantity
	}

	return executions
}

func eligibleRestingOrders(incoming Order, book []Order) []Order {
	eligible := make([]Order, 0, len(book))

	for _, resting := range book {
		if !isEligible(incoming, resting) {
			continue
		}
		eligible = append(eligible, resting)
	}

	return eligible
}

func isEligible(incoming, resting Order) bool {
	if resting.PlayerID != incoming.PlayerID {
		return false
	}
	if resting.Side != OppositeSide(incoming.Side) {
		return false
	}
	if resting.UserID == incoming.UserID {
		return false
	}
	if !resting.IsLive() || resting.RemainingQuantity() <= 0 {
		return false
	}

	return crosses(incoming, resting)
}

func crosses(incoming, resting Order) bool {
	if incoming.Side == SideBuy {
		return resting.Price <= incoming.Price
	}
	return resting.Price >= incoming.Price
}

func sortByPriceTimePriority(side OrderSide, orders []Order) {
	sort.SliceStable(orders, func(i, j int) bool {
		if orders[i].Price != orders[j].Price {
			if side == SideBuy {
				return orders[i].Price < orders[j].Price
			}
			return orders[i].Price > orders[j].Price
		}
		if !orders[i].CreatedAt.Equal(orders[j].CreatedAt) {
			return orders[i].CreatedAt.Before(orders[j].CreatedAt)
		}
		return orders[i].ID < orders[j].ID
	})
}
