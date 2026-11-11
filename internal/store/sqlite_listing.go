package store

import (
	"database/sql"

	"github.com/djshellshoxxx/circula/internal/domain"
)

const listingCols = `id,network_id,member_id,type,title,description,price,status,created_at,updated_at`

func (t *txImpl) InsertListing(l *domain.Listing) error {
	var price any
	if l.Price != nil {
		price = *l.Price
	}
	_, err := t.exec(`INSERT INTO listings(`+listingCols+`) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		l.ID, l.NetworkID, l.MemberID, string(l.Type), l.Title, l.Description, price,
		string(l.Status), ms(l.CreatedAt), ms(l.UpdatedAt))
	return err
}

func scanListing(r rowScanner) (*domain.Listing, error) {
	var l domain.Listing
	var typ, status string
	var price sql.NullInt64
	var created, updated int64
	if err := r.Scan(&l.ID, &l.NetworkID, &l.MemberID, &typ, &l.Title, &l.Description, &price,
		&status, &created, &updated); err != nil {
		return nil, err
	}
	l.Type, l.Status, l.Price = domain.ListingType(typ), domain.ListingStatus(status), ptrInt(price)
	l.CreatedAt, l.UpdatedAt = fromMS(created), fromMS(updated)
	return &l, nil
}

func (t *txImpl) GetListing(netID, id string) (*domain.Listing, error) {
	l, err := scanListing(t.row(`SELECT `+listingCols+` FROM listings WHERE network_id = ? AND id = ?`, netID, id))
	return l, notFound(err, "listing")
}

func (t *txImpl) UpdateListing(l *domain.Listing) error {
	var price any
	if l.Price != nil {
		price = *l.Price
	}
	_, err := t.exec(`UPDATE listings SET title=?, description=?, price=?, status=?, updated_at=?
		WHERE network_id = ? AND id = ?`, l.Title, l.Description, price, string(l.Status), ms(l.UpdatedAt), l.NetworkID, l.ID)
	return err
}

func (t *txImpl) ListListings(netID string, f ListingFilter) ([]*domain.Listing, error) {
	q := `SELECT ` + listingCols + ` FROM listings WHERE network_id = ?`
	args := []any{netID}
	if f.Type != "" {
		q += ` AND type = ?`
		args = append(args, string(f.Type))
	}
	if f.Status != "" {
		q += ` AND status = ?`
		args = append(args, string(f.Status))
	}
	if f.MemberID != "" {
		q += ` AND member_id = ?`
		args = append(args, f.MemberID)
	}
	if f.ActiveOwnersOnly {
		q += ` AND member_id IN (SELECT id FROM members WHERE network_id = ? AND status = 'active')`
		args = append(args, netID)
	}
	if f.Query != "" {
		// Plain substring test over folded text: ignores case and accents beyond ASCII
		// (domain.FoldName), with no LIKE escaping.
		term := domain.FoldName(f.Query)
		q += ` AND (instr(circula_fold(title), ?) > 0 OR instr(circula_fold(description), ?) > 0)`
		args = append(args, term, term)
	}
	if f.AfterID != "" {
		q += ` AND id < ?`
		args = append(args, f.AfterID)
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limitOf(f.Limit))
	rows, err := t.query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.Listing
	for rows.Next() {
		l, err := scanListing(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
