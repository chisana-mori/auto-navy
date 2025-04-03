declare module 'react-highlight-words' {
  import * as React from 'react';

  interface HighlighterProps {
    activeClassName?: string;
    activeIndex?: number;
    activeStyle?: React.CSSProperties;
    autoEscape?: boolean;
    className?: string;
    caseSensitive?: boolean;
    findChunks?: Function;
    highlightClassName?: string;
    highlightStyle?: React.CSSProperties;
    highlightTag?: string | React.ElementType;
    sanitize?: Function;
    searchWords: string[];
    textToHighlight: string;
    unhighlightClassName?: string;
    unhighlightStyle?: React.CSSProperties;
  }

  export default class Highlighter extends React.Component<HighlighterProps> {}
} 