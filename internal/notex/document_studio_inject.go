package notex

import (
	"context"
	"strings"
)

// StudioInjectionPrefixForLangGraph builds the same document blocks as chat/Studio when
// studio_document_ids are attached to a LangGraph run. Verifies each document belongs to
// the conversation's bound libraries and to the user.
func (s *Server) StudioInjectionPrefixForLangGraph(ctx context.Context, userID, conversationID int64, docIDs []int64) string {
	if s == nil || len(docIDs) == 0 || conversationID <= 0 {
		return ""
	}
	conv, uid := s.resolveConversationForStudioInject(ctx, userID, conversationID)
	if conv == nil || uid <= 0 {
		return ""
	}
	allowed := make(map[int64]struct{}, len(conv.LibraryIDs))
	for _, lid := range conv.LibraryIDs {
		allowed[lid] = struct{}{}
	}
	var blocks []string
	for _, docID := range docIDs {
		if docID <= 0 {
			continue
		}
		var doc *Document
		if s.store != nil {
			d, err := s.store.GetDocumentByIDForUser(ctx, uid, docID)
			if err != nil || d == nil {
				continue
			}
			doc = d
		} else {
			doc = s.getDocumentForStudio(ctx, docID)
			if doc == nil {
				continue
			}
		}
		if _, ok := allowed[doc.LibraryID]; !ok {
			continue
		}
		if block := s.studioDocumentBlock(doc); block != "" {
			blocks = append(blocks, block)
		}
	}
	return strings.Join(blocks, "\n\n")
}

func (s *Server) resolveConversationForStudioInject(ctx context.Context, userID, conversationID int64) (*Conversation, int64) {
	if userID > 0 {
		conv, err := s.getConversation(ctx, userID, conversationID)
		if err == nil && conv != nil {
			return conv, userID
		}
	}
	if s.store != nil {
		uid, conv, err := s.store.GetConversationWithOwner(ctx, conversationID)
		if err != nil || conv == nil || uid <= 0 {
			return nil, 0
		}
		if userID > 0 && uid != userID {
			return nil, 0
		}
		return conv, uid
	}
	s.conversationMu.RLock()
	defer s.conversationMu.RUnlock()
	for uid, list := range s.conversationsByUser {
		if userID > 0 && uid != userID {
			continue
		}
		for _, c := range list {
			if c != nil && c.ID == conversationID {
				return c, uid
			}
		}
	}
	return nil, 0
}
