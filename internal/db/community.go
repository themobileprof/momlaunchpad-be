package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

// CommunityPost is a feed post.
type CommunityPost struct {
	ID                string
	UserID            string
	Body              string
	IsAnonymous       bool
	Category          string
	Scope             string
	MedicalRelevance  string
	IsEvent           bool
	SafetyFlag        bool
	SpamScore         float32
	Status            string
	Country           *string
	StateProvince     *string
	City              *string
	LikeCount         int
	ReplyCount        int
	CreatedAt         time.Time
	UpdatedAt         time.Time
	AuthorName        *string
	AuthorPhotoURL      *string
	AuthorBadges        []string
	LikedByMe           bool
	ImageURLs           []string
}

// CommunityReply is a flat reply on a post.
type CommunityReply struct {
	ID           string
	PostID       string
	UserID       string
	Body         string
	IsAnonymous  bool
	LikeCount    int
	Status       string
	CreatedAt    time.Time
	AuthorName   *string
	AuthorPhotoURL *string
	AuthorBadges []string
	LikedByMe    bool
}

// CommunityEvent is a local event linked to a post.
type CommunityEvent struct {
	ID             string
	PostID         string
	EventType      *string
	Title          string
	Description    *string
	Venue          *string
	StartsAt       time.Time
	EndsAt         *time.Time
	Country        *string
	StateProvince  *string
	City           *string
	InterestedCount int
	InterestedByMe bool
	CreatedAt      time.Time
}

// CommunityNotification is an in-app notification.
type CommunityNotification struct {
	ID        string
	UserID    string
	Type      string
	Title     string
	Body      string
	Payload   map[string]any
	ReadAt    *time.Time
	CreatedAt time.Time
}

// CommunityReport is a moderation report.
type CommunityReport struct {
	ID         string
	ReporterID string
	TargetType string
	TargetID   string
	Reason     string
	Details    *string
	Status     string
	ReviewedBy *string
	ReviewedAt *time.Time
	CreatedAt  time.Time
}

// CompleteCommunityOnboarding saves location and marks community setup done.
func (db *DB) CompleteCommunityOnboarding(ctx context.Context, userID, countryCode, country, state, city string) error {
	query := `
		UPDATE users
		SET country = $1,
		    country_code = $2,
		    state_province = $3,
		    city = $4,
		    community_onboarding_completed_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`
	result, err := db.ExecContext(ctx, query, country, countryCode, state, city, userID)
	if err != nil {
		return fmt.Errorf("complete community onboarding: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// SetUserCommunityInterests replaces a user's interest selections (max 5 enforced by caller).
func (db *DB) SetUserCommunityInterests(ctx context.Context, userID string, keys []string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM community_user_interests WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("clear interests: %w", err)
	}
	for _, key := range keys {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO community_user_interests (user_id, interest_key) VALUES ($1, $2)`,
			userID, key,
		); err != nil {
			return fmt.Errorf("insert interest: %w", err)
		}
	}
	return tx.Commit()
}

// GetUserCommunityInterests returns selected interest keys.
func (db *DB) GetUserCommunityInterests(ctx context.Context, userID string) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT interest_key FROM community_user_interests WHERE user_id = $1 ORDER BY interest_key`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// GetUserBadges returns verified badge types for a user.
func (db *DB) GetUserBadges(ctx context.Context, userID string) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT badge_type FROM community_user_badges WHERE user_id = $1 ORDER BY badge_type`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var badges []string
	for rows.Next() {
		var b string
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		badges = append(badges, b)
	}
	return badges, rows.Err()
}

// GrantUserBadge assigns a verified badge (admin).
func (db *DB) GrantUserBadge(ctx context.Context, userID, badgeType, verifiedBy string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO community_user_badges (user_id, badge_type, verified_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, badge_type) DO NOTHING
	`, userID, badgeType, verifiedBy)
	return err
}

// RevokeUserBadge removes a badge (admin).
func (db *DB) RevokeUserBadge(ctx context.Context, userID, badgeType string) error {
	_, err := db.ExecContext(ctx,
		`DELETE FROM community_user_badges WHERE user_id = $1 AND badge_type = $2`,
		userID, badgeType,
	)
	return err
}

// CreateCommunityPost inserts a new post.
func (db *DB) CreateCommunityPost(ctx context.Context, post *CommunityPost) (*CommunityPost, error) {
	imageURLs := post.ImageURLs
	if imageURLs == nil {
		imageURLs = []string{}
	}

	query := `
		INSERT INTO community_posts (
			user_id, body, is_anonymous, category, scope, medical_relevance,
			is_event, safety_flag, spam_score, status, country, state_province, city,
			image_urls
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, like_count, reply_count, created_at, updated_at
	`
	err := db.QueryRowContext(ctx, query,
		post.UserID, post.Body, post.IsAnonymous, post.Category, post.Scope,
		post.MedicalRelevance, post.IsEvent, post.SafetyFlag, post.SpamScore,
		post.Status, post.Country, post.StateProvince, post.City, pq.Array(imageURLs),
	).Scan(&post.ID, &post.LikeCount, &post.ReplyCount, &post.CreatedAt, &post.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create post: %w", err)
	}
	return post, nil
}

// GetCommunityPostByID loads a single post with author metadata.
func (db *DB) GetCommunityPostByID(ctx context.Context, postID, viewerID string) (*CommunityPost, error) {
	query := `
		SELECT p.id, p.user_id, p.body, p.is_anonymous, p.category, p.scope,
		       p.medical_relevance, p.is_event, p.safety_flag, p.spam_score, p.status,
		       p.country, p.state_province, p.city, p.like_count, p.reply_count,
		       p.created_at, p.updated_at, p.image_urls,
		       u.display_name, u.profile_photo_url,
		       EXISTS(SELECT 1 FROM community_post_likes pl WHERE pl.post_id = p.id AND pl.user_id = $2)
		FROM community_posts p
		JOIN users u ON u.id = p.user_id
		WHERE p.id = $1 AND p.status = 'active'
	`
	post := &CommunityPost{}
	var authorName, authorPhoto sql.NullString
	var imageURLs pq.StringArray
	err := db.QueryRowContext(ctx, query, postID, viewerID).Scan(
		&post.ID, &post.UserID, &post.Body, &post.IsAnonymous, &post.Category, &post.Scope,
		&post.MedicalRelevance, &post.IsEvent, &post.SafetyFlag, &post.SpamScore, &post.Status,
		&post.Country, &post.StateProvince, &post.City, &post.LikeCount, &post.ReplyCount,
		&post.CreatedAt, &post.UpdatedAt, &imageURLs,
		&authorName, &authorPhoto, &post.LikedByMe,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if authorName.Valid {
		post.AuthorName = &authorName.String
	}
	if authorPhoto.Valid {
		post.AuthorPhotoURL = &authorPhoto.String
	}
	post.ImageURLs = []string(imageURLs)
	if !post.IsAnonymous {
		badges, _ := db.GetUserBadges(ctx, post.UserID)
		post.AuthorBadges = badges
	}
	return post, nil
}

// ListCommunityFeed returns posts for a feed filter.
func (db *DB) ListCommunityFeed(ctx context.Context, viewerID, filter string, limit int, cursor *time.Time) ([]CommunityPost, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	user, err := db.GetUserByID(ctx, viewerID)
	if err != nil {
		return nil, err
	}

	interests, _ := db.GetUserCommunityInterests(ctx, viewerID)

	var sb strings.Builder
	sb.WriteString(`
		SELECT p.id, p.user_id, p.body, p.is_anonymous, p.category, p.scope,
		       p.medical_relevance, p.is_event, p.safety_flag, p.spam_score, p.status,
		       p.country, p.state_province, p.city, p.like_count, p.reply_count,
		       p.created_at, p.updated_at, p.image_urls,
		       u.display_name, u.profile_photo_url,
		       EXISTS(SELECT 1 FROM community_post_likes pl WHERE pl.post_id = p.id AND pl.user_id = $1)
		FROM community_posts p
		JOIN users u ON u.id = p.user_id
		WHERE p.status = 'active'
		  AND NOT EXISTS (SELECT 1 FROM community_hidden_posts hp WHERE hp.user_id = $1 AND hp.post_id = p.id)
		  AND NOT EXISTS (
		      SELECT 1 FROM community_blocks b
		      WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id)
		         OR (b.blocker_id = p.user_id AND b.blocked_id = $1)
		  )
	`)

	args := []any{viewerID}
	argN := 2

	switch filter {
	case "my_posts":
		sb.WriteString(fmt.Sprintf(" AND p.user_id = $%d", argN))
		args = append(args, viewerID)
		argN++
	case "events":
		sb.WriteString(" AND p.is_event = TRUE")
	case "nearby":
		if user.Country != nil && *user.Country != "" {
			sb.WriteString(fmt.Sprintf(" AND p.country = $%d", argN))
			args = append(args, *user.Country)
			argN++
		}
		if user.StateProvince != nil && *user.StateProvince != "" {
			sb.WriteString(fmt.Sprintf(" AND p.state_province = $%d", argN))
			args = append(args, *user.StateProvince)
			argN++
		}
		if user.City != nil && *user.City != "" {
			sb.WriteString(fmt.Sprintf(" AND p.city = $%d", argN))
			args = append(args, *user.City)
			argN++
		}
	default: // for_you
		if len(interests) > 0 {
			sb.WriteString(fmt.Sprintf(`
			 AND (
			   p.category = ANY($%d)
			   OR EXISTS (
			     SELECT 1 FROM community_follows f
			     WHERE f.follower_id = $1 AND f.following_id = p.user_id
			   )
			 )`, argN))
			args = append(args, pq.Array(interests))
			argN++
		}
	}

	if cursor != nil {
		sb.WriteString(fmt.Sprintf(" AND p.created_at < $%d", argN))
		args = append(args, *cursor)
		argN++
	}

	sb.WriteString(fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d", argN))
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []CommunityPost
	for rows.Next() {
		var post CommunityPost
		var authorName, authorPhoto sql.NullString
		var imageURLs pq.StringArray
		if err := rows.Scan(
			&post.ID, &post.UserID, &post.Body, &post.IsAnonymous, &post.Category, &post.Scope,
			&post.MedicalRelevance, &post.IsEvent, &post.SafetyFlag, &post.SpamScore, &post.Status,
			&post.Country, &post.StateProvince, &post.City, &post.LikeCount, &post.ReplyCount,
			&post.CreatedAt, &post.UpdatedAt, &imageURLs,
			&authorName, &authorPhoto, &post.LikedByMe,
		); err != nil {
			return nil, err
		}
		if authorName.Valid {
			post.AuthorName = &authorName.String
		}
		if authorPhoto.Valid {
			post.AuthorPhotoURL = &authorPhoto.String
		}
		post.ImageURLs = []string(imageURLs)
		if !post.IsAnonymous {
			badges, _ := db.GetUserBadges(ctx, post.UserID)
			post.AuthorBadges = badges
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

// CreateCommunityReply adds a flat reply.
func (db *DB) CreateCommunityReply(ctx context.Context, reply *CommunityReply) (*CommunityReply, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO community_replies (post_id, user_id, body, is_anonymous, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, like_count, created_at
	`
	err = tx.QueryRowContext(ctx, query,
		reply.PostID, reply.UserID, reply.Body, reply.IsAnonymous, reply.Status,
	).Scan(&reply.ID, &reply.LikeCount, &reply.CreatedAt)
	if err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE community_posts SET reply_count = reply_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		reply.PostID,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return reply, nil
}

// ListCommunityReplies returns replies for a post.
func (db *DB) ListCommunityReplies(ctx context.Context, postID, viewerID string) ([]CommunityReply, error) {
	query := `
		SELECT r.id, r.post_id, r.user_id, r.body, r.is_anonymous, r.like_count, r.status, r.created_at,
		       u.display_name, u.profile_photo_url,
		       EXISTS(SELECT 1 FROM community_reply_likes rl WHERE rl.reply_id = r.id AND rl.user_id = $2)
		FROM community_replies r
		JOIN users u ON u.id = r.user_id
		WHERE r.post_id = $1 AND r.status = 'active'
		ORDER BY r.created_at ASC
	`
	rows, err := db.QueryContext(ctx, query, postID, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var replies []CommunityReply
	for rows.Next() {
		var r CommunityReply
		var authorName, authorPhoto sql.NullString
		if err := rows.Scan(
			&r.ID, &r.PostID, &r.UserID, &r.Body, &r.IsAnonymous, &r.LikeCount, &r.Status, &r.CreatedAt,
			&authorName, &authorPhoto, &r.LikedByMe,
		); err != nil {
			return nil, err
		}
		if authorName.Valid {
			r.AuthorName = &authorName.String
		}
		if authorPhoto.Valid {
			r.AuthorPhotoURL = &authorPhoto.String
		}
		if !r.IsAnonymous {
			badges, _ := db.GetUserBadges(ctx, r.UserID)
			r.AuthorBadges = badges
		}
		replies = append(replies, r)
	}
	return replies, rows.Err()
}

// TogglePostLike toggles like and returns new liked state and count.
func (db *DB) TogglePostLike(ctx context.Context, postID, userID string) (liked bool, count int, err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM community_post_likes WHERE post_id = $1 AND user_id = $2)`,
		postID, userID,
	).Scan(&exists)
	if err != nil {
		return false, 0, err
	}

	if exists {
		if _, err = tx.ExecContext(ctx,
			`DELETE FROM community_post_likes WHERE post_id = $1 AND user_id = $2`, postID, userID); err != nil {
			return false, 0, err
		}
		if _, err = tx.ExecContext(ctx,
			`UPDATE community_posts SET like_count = GREATEST(like_count - 1, 0) WHERE id = $1`, postID); err != nil {
			return false, 0, err
		}
		liked = false
	} else {
		if _, err = tx.ExecContext(ctx,
			`INSERT INTO community_post_likes (post_id, user_id) VALUES ($1, $2)`, postID, userID); err != nil {
			return false, 0, err
		}
		if _, err = tx.ExecContext(ctx,
			`UPDATE community_posts SET like_count = like_count + 1 WHERE id = $1`, postID); err != nil {
			return false, 0, err
		}
		liked = true
	}

	err = tx.QueryRowContext(ctx, `SELECT like_count FROM community_posts WHERE id = $1`, postID).Scan(&count)
	if err != nil {
		return liked, 0, err
	}
	return liked, count, tx.Commit()
}

// ToggleReplyLike toggles reply like.
func (db *DB) ToggleReplyLike(ctx context.Context, replyID, userID string) (liked bool, count int, err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM community_reply_likes WHERE reply_id = $1 AND user_id = $2)`,
		replyID, userID,
	).Scan(&exists)
	if err != nil {
		return false, 0, err
	}

	if exists {
		_, err = tx.ExecContext(ctx,
			`DELETE FROM community_reply_likes WHERE reply_id = $1 AND user_id = $2`, replyID, userID)
		if err != nil {
			return false, 0, err
		}
		_, err = tx.ExecContext(ctx,
			`UPDATE community_replies SET like_count = GREATEST(like_count - 1, 0) WHERE id = $1`, replyID)
		liked = false
	} else {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO community_reply_likes (reply_id, user_id) VALUES ($1, $2)`, replyID, userID)
		if err != nil {
			return false, 0, err
		}
		_, err = tx.ExecContext(ctx,
			`UPDATE community_replies SET like_count = like_count + 1 WHERE id = $1`, replyID)
		liked = true
	}
	if err != nil {
		return false, 0, err
	}

	err = tx.QueryRowContext(ctx, `SELECT like_count FROM community_replies WHERE id = $1`, replyID).Scan(&count)
	if err != nil {
		return liked, 0, err
	}
	return liked, count, tx.Commit()
}

// CreateCommunityEvent links an event to a post.
func (db *DB) CreateCommunityEvent(ctx context.Context, event *CommunityEvent) (*CommunityEvent, error) {
	query := `
		INSERT INTO community_events (
			post_id, event_type, title, description, venue, starts_at, ends_at, country, state_province, city
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, interested_count, created_at
	`
	err := db.QueryRowContext(ctx, query,
		event.PostID, event.EventType, event.Title, event.Description, event.Venue,
		event.StartsAt, event.EndsAt, event.Country, event.StateProvince, event.City,
	).Scan(&event.ID, &event.InterestedCount, &event.CreatedAt)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// GetCommunityEventByPostID loads event details.
func (db *DB) GetCommunityEventByPostID(ctx context.Context, postID, viewerID string) (*CommunityEvent, error) {
	query := `
		SELECT e.id, e.post_id, e.event_type, e.title, e.description, e.venue, e.starts_at, e.ends_at,
		       e.country, e.state_province, e.city, e.interested_count, e.created_at,
		       EXISTS(SELECT 1 FROM community_event_interests ei WHERE ei.event_id = e.id AND ei.user_id = $2)
		FROM community_events e
		WHERE e.post_id = $1
	`
	event := &CommunityEvent{}
	err := db.QueryRowContext(ctx, query, postID, viewerID).Scan(
		&event.ID, &event.PostID, &event.EventType, &event.Title, &event.Description, &event.Venue,
		&event.StartsAt, &event.EndsAt, &event.Country, &event.StateProvince, &event.City,
		&event.InterestedCount, &event.CreatedAt, &event.InterestedByMe,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return event, nil
}

// ToggleEventInterest toggles RSVP-style interest.
func (db *DB) ToggleEventInterest(ctx context.Context, eventID, userID string) (interested bool, count int, err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM community_event_interests WHERE event_id = $1 AND user_id = $2)`,
		eventID, userID,
	).Scan(&exists)
	if err != nil {
		return false, 0, err
	}

	if exists {
		_, _ = tx.ExecContext(ctx,
			`DELETE FROM community_event_interests WHERE event_id = $1 AND user_id = $2`, eventID, userID)
		_, _ = tx.ExecContext(ctx,
			`UPDATE community_events SET interested_count = GREATEST(interested_count - 1, 0) WHERE id = $1`, eventID)
		interested = false
	} else {
		_, _ = tx.ExecContext(ctx,
			`INSERT INTO community_event_interests (event_id, user_id) VALUES ($1, $2)`, eventID, userID)
		_, _ = tx.ExecContext(ctx,
			`UPDATE community_events SET interested_count = interested_count + 1 WHERE id = $1`, eventID)
		interested = true
	}

	err = tx.QueryRowContext(ctx, `SELECT interested_count FROM community_events WHERE id = $1`, eventID).Scan(&count)
	if err != nil {
		return interested, 0, err
	}

	if err := syncCommunityEventReminderTx(ctx, tx, userID, eventID, interested); err != nil {
		return interested, 0, err
	}

	return interested, count, tx.Commit()
}

func syncCommunityEventReminderTx(ctx context.Context, tx *sql.Tx, userID, eventID string, interested bool) error {
	if !interested {
		_, err := tx.ExecContext(ctx,
			`DELETE FROM reminders WHERE user_id = $1 AND community_event_id = $2`,
			userID, eventID,
		)
		return err
	}

	var title string
	var description, venue, city, stateProvince, country sql.NullString
	var startsAt time.Time
	err := tx.QueryRowContext(ctx, `
		SELECT title, description, venue, starts_at, city, state_province, country
		FROM community_events
		WHERE id = $1
	`, eventID).Scan(&title, &description, &venue, &startsAt, &city, &stateProvince, &country)
	if err != nil {
		return err
	}

	reminderDescription := buildCommunityEventReminderDescription(
		description, venue, city, stateProvince, country,
	)

	_, err = tx.ExecContext(ctx,
		`DELETE FROM reminders WHERE user_id = $1 AND community_event_id = $2`,
		userID, eventID,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO reminders (user_id, title, description, reminder_time, is_completed, community_event_id)
		VALUES ($1, $2, $3, $4, FALSE, $5)
	`, userID, title, reminderDescription, startsAt, eventID)
	return err
}

func buildCommunityEventReminderDescription(
	description, venue, city, stateProvince, country sql.NullString,
) *string {
	var parts []string
	if description.Valid && strings.TrimSpace(description.String) != "" {
		parts = append(parts, strings.TrimSpace(description.String))
	}
	if venue.Valid && strings.TrimSpace(venue.String) != "" {
		parts = append(parts, "Venue: "+strings.TrimSpace(venue.String))
	}

	locationParts := make([]string, 0, 3)
	if city.Valid && strings.TrimSpace(city.String) != "" {
		locationParts = append(locationParts, strings.TrimSpace(city.String))
	}
	if stateProvince.Valid && strings.TrimSpace(stateProvince.String) != "" {
		locationParts = append(locationParts, strings.TrimSpace(stateProvince.String))
	}
	if country.Valid && strings.TrimSpace(country.String) != "" {
		locationParts = append(locationParts, strings.TrimSpace(country.String))
	}
	if len(locationParts) > 0 {
		parts = append(parts, strings.Join(locationParts, ", "))
	}

	parts = append(parts, "Added from MomLaunchpad Community")
	text := strings.Join(parts, "\n\n")
	return &text
}

// FollowUser creates a follow relationship.
func (db *DB) FollowUser(ctx context.Context, followerID, followingID string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO community_follows (follower_id, following_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		followerID, followingID,
	)
	return err
}

// UnfollowUser removes a follow.
func (db *DB) UnfollowUser(ctx context.Context, followerID, followingID string) error {
	_, err := db.ExecContext(ctx,
		`DELETE FROM community_follows WHERE follower_id = $1 AND following_id = $2`,
		followerID, followingID,
	)
	return err
}

// BlockUser blocks another user.
func (db *DB) BlockUser(ctx context.Context, blockerID, blockedID string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO community_blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		blockerID, blockedID,
	)
	return err
}

// HidePost hides a post for a user.
func (db *DB) HidePost(ctx context.Context, userID, postID string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO community_hidden_posts (user_id, post_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, postID,
	)
	return err
}

// CreateCommunityReport files a moderation report.
func (db *DB) CreateCommunityReport(ctx context.Context, report *CommunityReport) (*CommunityReport, error) {
	query := `
		INSERT INTO community_reports (reporter_id, target_type, target_id, reason, details)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, created_at
	`
	err := db.QueryRowContext(ctx, query,
		report.ReporterID, report.TargetType, report.TargetID, report.Reason, report.Details,
	).Scan(&report.ID, &report.Status, &report.CreatedAt)
	if err != nil {
		return nil, err
	}
	return report, nil
}

// ListCommunityReports lists reports for admin (optionally filtered by status).
func (db *DB) ListCommunityReports(ctx context.Context, status string, limit int) ([]CommunityReport, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, reporter_id, target_type, target_id, reason, details, status,
		       reviewed_by, reviewed_at, created_at
		FROM community_reports
		WHERE ($1 = '' OR status = $1)
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := db.QueryContext(ctx, query, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []CommunityReport
	for rows.Next() {
		var r CommunityReport
		var details sql.NullString
		var reviewedBy sql.NullString
		var reviewedAt sql.NullTime
		if err := rows.Scan(
			&r.ID, &r.ReporterID, &r.TargetType, &r.TargetID, &r.Reason, &details,
			&r.Status, &reviewedBy, &reviewedAt, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		if details.Valid {
			r.Details = &details.String
		}
		if reviewedBy.Valid {
			r.ReviewedBy = &reviewedBy.String
		}
		if reviewedAt.Valid {
			r.ReviewedAt = &reviewedAt.Time
		}
		reports = append(reports, r)
	}
	return reports, rows.Err()
}

// UpdateCommunityReportStatus updates report moderation state.
func (db *DB) UpdateCommunityReportStatus(ctx context.Context, reportID, status, reviewerID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE community_reports
		SET status = $1, reviewed_by = $2, reviewed_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`, status, reviewerID, reportID)
	return err
}

// UpdateCommunityPostStatus moderates a post (admin or auto-moderation).
func (db *DB) UpdateCommunityPostStatus(ctx context.Context, postID, status string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE community_posts SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		status, postID,
	)
	return err
}

// CreateCommunityNotification inserts an in-app notification.
func (db *DB) CreateCommunityNotification(ctx context.Context, n *CommunityNotification) error {
	payload, err := json.Marshal(n.Payload)
	if err != nil {
		payload = []byte("{}")
	}
	return db.QueryRowContext(ctx, `
		INSERT INTO community_notifications (user_id, type, title, body, payload)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, n.UserID, n.Type, n.Title, n.Body, payload).Scan(&n.ID, &n.CreatedAt)
}

// ListCommunityNotifications returns notifications for a user.
func (db *DB) ListCommunityNotifications(ctx context.Context, userID string, limit int) ([]CommunityNotification, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, user_id, type, title, body, payload, read_at, created_at
		FROM community_notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CommunityNotification
	for rows.Next() {
		var n CommunityNotification
		var payload []byte
		var readAt sql.NullTime
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &payload, &readAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(payload, &n.Payload)
		if readAt.Valid {
			n.ReadAt = &readAt.Time
		}
		items = append(items, n)
	}
	return items, rows.Err()
}

// MarkNotificationRead marks a notification read.
func (db *DB) MarkNotificationRead(ctx context.Context, notificationID, userID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE community_notifications SET read_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2
	`, notificationID, userID)
	return err
}

// GetPostAuthorID returns the author of a post.
func (db *DB) GetPostAuthorID(ctx context.Context, postID string) (string, error) {
	var authorID string
	err := db.QueryRowContext(ctx, `SELECT user_id FROM community_posts WHERE id = $1`, postID).Scan(&authorID)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	return authorID, err
}
