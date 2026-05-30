import { useEffect, useRef } from 'react';

interface UseAutoResizeTextareaOptions {
  value: string;
  minHeight?: number;
  maxHeight?: number;
}

export function useAutoResizeTextarea({
  value,
  minHeight = 120,
  maxHeight = 800
}: UseAutoResizeTextareaOptions) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    // Reset height to allow shrinking
    textarea.style.height = 'auto';
    
    // Calculate new height based on scroll height
    const scrollHeight = textarea.scrollHeight;
    const newHeight = Math.min(Math.max(scrollHeight, minHeight), maxHeight);
    
    // Apply the new height
    textarea.style.height = `${newHeight}px`;
    
    // If content exceeds maxHeight, allow internal scrolling
    textarea.style.overflowY = scrollHeight > maxHeight ? 'auto' : 'hidden';
  }, [value, minHeight, maxHeight]);

  return textareaRef;
}