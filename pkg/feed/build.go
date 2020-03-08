package feed

import (
	"context"
	"fmt"
	"strconv"
	"time"

	itunes "github.com/eduncan911/podcast"
	"github.com/pkg/errors"

	"github.com/mxpv/podsync/pkg/config"
	"github.com/mxpv/podsync/pkg/model"
)

type urlProvider interface {
	URL(ctx context.Context, ns string, fileName string) (string, error)
}

func Build(ctx context.Context, feed *model.Feed, cfg *config.Feed, provider urlProvider) (*itunes.Podcast, error) {
	const (
		podsyncGenerator = "Podsync generator (support us at https://github.com/mxpv/podsync)"
		defaultCategory  = "TV & Film"
	)

	now := time.Now().UTC()

	p := itunes.New(feed.Title, feed.ItemURL, feed.Description, &feed.PubDate, &now)
	p.Generator = podsyncGenerator
	p.AddSubTitle(feed.Title)
	p.AddCategory(defaultCategory, nil)
	p.AddImage(feed.CoverArt)
	p.IAuthor = feed.Title
	p.AddSummary(feed.Description)

	if feed.Explicit {
		p.IExplicit = "yes"
	} else {
		p.IExplicit = "no"
	}

	if feed.Language != "" {
		p.Language = feed.Language
	}

	for i, episode := range feed.Episodes {
		if episode.Status != model.EpisodeDownloaded {
			// Skip episodes that are not yet downloaded
			continue
		}

		item := itunes.Item{
			GUID:        episode.ID,
			Link:        episode.VideoURL,
			Title:       episode.Title,
			Description: episode.Description,
			ISubtitle:   episode.Title,
			IOrder:      strconv.Itoa(i),
		}

		pubDate := episode.PubDate
		if pubDate.IsZero() {
			pubDate = now
		}

		item.AddPubDate(&pubDate)

		item.AddSummary(episode.Description)
		item.AddImage(episode.Thumbnail)
		item.AddDuration(episode.Duration)

		enclosureType := itunes.MP4
		if feed.Format == model.FormatAudio {
			enclosureType = itunes.MP4
		}

		episodeName := EpisodeName(cfg, episode)
		downloadURL, err := provider.URL(ctx, cfg.ID, episodeName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to obtain download URL for: %s", episodeName)
		}

		item.AddEnclosure(downloadURL, enclosureType, episode.Size)

		// p.AddItem requires description to be not empty, use workaround
		if item.Description == "" {
			item.Description = " "
		}

		if feed.Explicit {
			item.IExplicit = "yes"
		} else {
			item.IExplicit = "no"
		}

		if _, err := p.AddItem(item); err != nil {
			return nil, errors.Wrapf(err, "failed to add item to podcast (id %q)", episode.ID)
		}
	}

	return &p, nil
}

func EpisodeName(feedConfig *config.Feed, episode *model.Episode) string {
	ext := "mp4"
	if feedConfig.Format == model.FormatAudio {
		ext = "mp3"
	}

	return fmt.Sprintf("%s.%s", episode.ID, ext)
}
