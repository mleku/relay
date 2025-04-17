package relay

import (
	"bytes"
	"fmt"
	"strings"

	"relay.mleku.dev/bech32encoding"
	"relay.mleku.dev/chk"
	"relay.mleku.dev/context"
	"relay.mleku.dev/event"
	"relay.mleku.dev/filter"
	"relay.mleku.dev/hex"
	"relay.mleku.dev/kind"
	"relay.mleku.dev/kinds"
	"relay.mleku.dev/log"
	"relay.mleku.dev/tag"
)

func (s *Server) Init() {
	var err error
	s.configurationMx.Lock()
	defer s.configurationMx.Unlock()
	for _, src := range s.configuration.Owners {
		if len(src) < 1 {
			continue
		}
		dst := make([]byte, len(src)/2)
		if _, err = hex.DecBytes(dst, []byte(src)); err != nil {
			if dst, err = bech32encoding.NpubToBytes([]byte(src)); chk.E(err) {
				continue
			}
		}
		s.owners = append(s.owners, dst)
	}
	if len(s.owners) > 0 {
		log.T.C(func() string {
			ownerIds := make([]string, len(s.owners))
			for i, npub := range s.owners {
				ownerIds[i] = hex.Enc(npub)
			}
			owners := strings.Join(ownerIds, ",")
			return fmt.Sprintf("owners %s", owners)
		})
		s.ZeroLists()
		s.CheckOwnerLists(context.Bg())
	}
}

func (s *Server) ZeroLists() {
	s.Lock()
	defer s.Unlock()
	s.Followed = make(map[string]struct{})
	s.ownersFollowed = make(map[string]struct{})
	s.OwnersFollowLists = s.OwnersFollowLists[:0]
	s.Muted = make(map[string]struct{})
	s.OwnersMuteLists = s.OwnersMuteLists[:0]
}

// CheckOwnerLists regenerates the owner follow and mute lists if they are empty.
//
// It also adds the followed npubs of the follows.
func (s *Server) CheckOwnerLists(c context.T) {
	if len(s.owners) > 0 {
		s.Lock()
		defer s.Unlock()
		var err error
		var evs []*event.T
		// need to search DB for moderator npub follow lists, followed npubs are allowed access.
		if len(s.Followed) < 1 {
			// add the owners themselves of course
			for i := range s.owners {
				s.Followed[string(s.owners[i])] = struct{}{}
			}
			log.D.Ln("regenerating owners follow lists")
			if evs, err = s.Store.QueryEvents(c,
				&filter.T{Authors: tag.New(s.owners...),
					Kinds: kinds.New(kind.FollowList)}); chk.E(err) {
			}
			for _, ev := range evs {
				s.OwnersFollowLists = append(s.OwnersFollowLists, ev.Id)
				for _, t := range ev.Tags.ToSliceOfTags() {
					if bytes.Equal(t.Key(), []byte("p")) {
						var p []byte
						if p, err = hex.Dec(string(t.Value())); chk.E(err) {
							continue
						}
						s.Followed[string(p)] = struct{}{}
						s.ownersFollowed[string(p)] = struct{}{}
					}
				}
			}
			evs = evs[:0]
			// next, search for the follow lists of all on the follow list
			log.D.Ln("searching for owners follows follow lists")
			var followed []string
			for f := range s.Followed {
				followed = append(followed, f)
			}
			if evs, err = s.Store.QueryEvents(c,
				&filter.T{Authors: tag.New(followed...),
					Kinds: kinds.New(kind.FollowList)}); chk.E(err) {
			}
			for _, ev := range evs {
				// we want to protect the follow lists of users as well so they also cannot be
				// deleted, only replaced.
				s.OwnersFollowLists = append(s.OwnersFollowLists, ev.Id)
				for _, t := range ev.Tags.ToSliceOfTags() {
					if bytes.Equal(t.Key(), []byte("p")) {
						var p []byte
						if p, err = hex.Dec(string(t.Value())); err != nil {
							continue
						}
						s.Followed[string(p)] = struct{}{}
					}
				}
			}
			evs = evs[:0]
		}
		if len(s.Muted) < 1 {
			log.D.Ln("regenerating owners mute lists")
			s.Muted = make(map[string]struct{})
			if evs, err = s.Store.QueryEvents(c,
				&filter.T{Authors: tag.New(s.owners...),
					Kinds: kinds.New(kind.MuteList)}); chk.E(err) {
			}
			for _, ev := range evs {
				s.OwnersMuteLists = append(s.OwnersMuteLists, ev.Id)
				for _, t := range ev.Tags.ToSliceOfTags() {
					if bytes.Equal(t.Key(), []byte("p")) {
						var p []byte
						if p, err = hex.Dec(string(t.Value())); chk.E(err) {
							continue
						}
						s.Muted[string(p)] = struct{}{}
					}
				}
			}
			evs = evs[:0]
		}
		// remove muted from the followed list
		for m := range s.Muted {
			for f := range s.Followed {
				if f == m {
					// delete muted element from Followed list
					delete(s.Followed, m)
				}
			}
		}
		log.I.F("%d allowed npubs, %d blocked", len(s.Followed), len(s.Muted))
	}
}
