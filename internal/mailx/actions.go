package mailx

import (
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"glossmail/internal/store"
)

// FlagPatch updates seen/flagged flags.
type FlagPatch struct {
	Seen    *bool
	Flagged *bool
}

func uidSet(uids []uint32) imap.UIDSet {
	set := imap.UIDSet{}
	for _, u := range uids {
		set.AddNum(imap.UID(u))
	}
	return set
}

// SetFlags applies flag changes to given uids.
func SetFlags(a *store.Account, mailbox string, uids []uint32, patch FlagPatch) error {
	set := uidSet(uids)
	err := withClient(a, func(c *imapclient.Client) error {
		if _, err := c.Select(mailbox, nil).Wait(); err != nil {
			return errf(404, "邮箱不存在：%s", mailbox)
		}
		for _, p := range []struct {
			val  *bool
			flag imap.Flag
		}{{patch.Seen, "\\Seen"}, {patch.Flagged, "\\Flagged"}} {
			if p.val == nil {
				continue
			}
			op := imap.StoreFlagsAdd
			if !*p.val {
				op = imap.StoreFlagsDel
			}
			sf := imap.StoreFlags{Op: op, Flags: []imap.Flag{p.flag}, Silent: true}
			if err := c.Store(&set, &sf, nil).Close(); err != nil {
				return errf(502, "更新标记失败：%s", explain(err))
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// mirror into the cache so the UI is instant
	if doc := loadEnv(a.ID, mailbox); doc != nil {
		inSet := map[uint32]bool{}
		for _, u := range uids {
			inSet[u] = true
		}
		for _, m := range doc.Messages {
			if !inSet[m.UID] {
				continue
			}
			if patch.Seen != nil {
				m.Flags.Seen = *patch.Seen
			}
			if patch.Flagged != nil {
				m.Flags.Flagged = *patch.Flagged
			}
		}
		saveEnv(a.ID, mailbox, doc)
	}
	return nil
}

// MoveMessages moves uids from one folder to another.
func MoveMessages(a *store.Account, from string, uids []uint32, dest string) error {
	set := uidSet(uids)
	err := withClient(a, func(c *imapclient.Client) error {
		if _, err := c.Select(from, nil).Wait(); err != nil {
			return errf(404, "邮箱不存在：%s", from)
		}
		if _, err := c.Move(&set, dest).Wait(); err != nil {
			return errf(502, "移动失败：%s", explain(err))
		}
		return nil
	})
	if err != nil {
		return err
	}
	invalidateLocal(a.ID, from, uids, false)
	invalidateLocal(a.ID, dest, uids, true)
	return nil
}

// DeleteMessages moves to Trash, or expunges when already in Trash.
func DeleteMessages(a *store.Account, mailbox string, uids []uint32) error {
	trashBox, _ := FindMailboxByKind(a, "trash")
	if trashBox != nil && trashBox.Path == mailbox {
		return expunge(a, mailbox, uids)
	}
	if trashBox != nil {
		if err := MoveMessages(a, mailbox, uids, trashBox.Path); err == nil {
			return nil
		}
	}
	return expunge(a, mailbox, uids)
}

func expunge(a *store.Account, mailbox string, uids []uint32) error {
	set := uidSet(uids)
	err := withClient(a, func(c *imapclient.Client) error {
		if _, err := c.Select(mailbox, nil).Wait(); err != nil {
			return errf(404, "邮箱不存在：%s", mailbox)
		}
		if err := c.Store(&set, &imap.StoreFlags{Op: imap.StoreFlagsAdd, Flags: []imap.Flag{"\\Deleted"}, Silent: true}, nil).Close(); err != nil {
			return errf(502, "删除失败：%s", explain(err))
		}
		if err := c.Expunge().Close(); err != nil {
			return errf(502, "清理失败：%s", explain(err))
		}
		return nil
	})
	if err != nil {
		return err
	}
	invalidateLocal(a.ID, mailbox, uids, false)
	return nil
}

// ArchiveMessages moves messages to the Archive (or All) folder.
func ArchiveMessages(a *store.Account, mailbox string, uids []uint32) error {
	box, _ := FindMailboxByKind(a, "archive")
	if box == nil {
		box, _ = FindMailboxByKind(a, "all")
	}
	if box == nil {
		return errf(400, "未找到归档邮箱")
	}
	return MoveMessages(a, mailbox, uids, box.Path)
}

// AppendToMailbox uploads a raw message to a folder.
func AppendToMailbox(a *store.Account, mailbox string, raw []byte) error {
	return withClient(a, func(c *imapclient.Client) error {
		cmd := c.Append(mailbox, int64(len(raw)), &imap.AppendOptions{
			Flags: []imap.Flag{"\\Seen"}, Time: time.Now(),
		})
		if _, err := cmd.Write(raw); err != nil {
			return errf(502, "写入已发送失败：%s", explain(err))
		}
		if err := cmd.Close(); err != nil {
			return errf(502, "写入已发送失败：%s", explain(err))
		}
		if _, err := cmd.Wait(); err != nil {
			return errf(502, "写入已发送失败：%s", explain(err))
		}
		return nil
	})
}

func invalidateLocal(accountID, mailbox string, uids []uint32, bump bool) {
	doc := loadEnv(accountID, mailbox)
	if doc == nil {
		return
	}
	set := map[uint32]bool{}
	for _, u := range uids {
		set[u] = true
	}
	kept := doc.Messages[:0]
	for _, m := range doc.Messages {
		if !set[m.UID] {
			kept = append(kept, m)
		}
	}
	doc.Messages = kept
	if bump {
		doc.UIDNext = 0 // force full re-sync of destination
	}
	saveEnv(accountID, mailbox, doc)
}

