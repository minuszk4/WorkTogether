import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'markdown',
  standalone: true
})
export class MarkdownPipe implements PipeTransform {
  transform(value: string): string {
    if (!value) return '';

    // Replace HTML tags to prevent XSS (very basic)
    let safeValue = value.replace(/&/g, '&amp;')
                         .replace(/</g, '&lt;')
                         .replace(/>/g, '&gt;');

    // Bold: **text**
    safeValue = safeValue.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
    
    // Italic: *text*
    safeValue = safeValue.replace(/\*(.*?)\*/g, '<em>$1</em>');
    
    // Code block: `code`
    safeValue = safeValue.replace(/`(.*?)`/g, '<code>$1</code>');
    
    // Strikethrough: ~~text~~
    safeValue = safeValue.replace(/~~(.*?)~~/g, '<del>$1</del>');

    return safeValue;
  }
}
