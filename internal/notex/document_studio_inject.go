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
		s.logger.Printf("[studio-inject] early return: s.nil=%v, len(docIDs)=%d, conversationID=%d", s == nil, len(docIDs), conversationID)
		return ""
	}
	conv, uid := s.resolveConversationForStudioInject(ctx, userID, conversationID)
	if conv == nil || uid <= 0 {
		s.logger.Printf("[studio-inject] resolve conversation failed: conv.nil=%v, uid=%d", conv == nil, uid)
		return ""
	}
	s.logger.Printf("[studio-inject] resolved conversation: uid=%d, conv_id=%d, library_ids=%v", uid, conv.ID, conv.LibraryIDs)
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
				s.logger.Printf("[studio-inject] doc %d not found for user %d: err=%v", docID, uid, err)
				continue
			}
			doc = d
		} else {
			doc = s.getDocumentForStudio(ctx, docID)
			if doc == nil {
				s.logger.Printf("[studio-inject] doc %d not found in memory", docID)
				continue
			}
		}
		if _, ok := allowed[doc.LibraryID]; !ok {
			s.logger.Printf("[studio-inject] doc %d library %d not in allowed list %v", doc.ID, doc.LibraryID, allowed)
			continue
		}
		block := s.studioDocumentBlock(doc)
		s.logger.Printf("[studio-inject] doc %d (%s): status=%s, block_len=%d", doc.ID, doc.OriginalName, doc.ExtractionStatus, len(block))
		if block != "" {
			blocks = append(blocks, block)
		}
	}
	result := strings.Join(blocks, "\n\n")
	s.logger.Printf("[studio-inject] final result: %d blocks, total_len=%d", len(blocks), len(result))
	return result
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
