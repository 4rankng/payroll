import React from "react";
import { cn } from "@/lib/utils";

interface AnimatedHamburgerProps {
  isOpen: boolean;
  className?: string;
  size?: number;
  strokeWidth?: number;
  color?: string;
}

export function AnimatedHamburger({ 
  isOpen, 
  className, 
  size = 20, 
  strokeWidth = 2,
  color = "currentColor" 
}: AnimatedHamburgerProps) {
  return (
    <div className={cn("flex items-center justify-center", className)}>
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        className="transition-all duration-300 ease-in-out"
      >
        {/* Top line */}
        <path
          d="M3 6h18"
          stroke={color}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          className={cn(
            "transition-all duration-300 ease-in-out origin-center",
            !isOpen 
              ? "rotate-0 translate-y-0 opacity-100" 
              : "rotate-45 translate-y-[6px] opacity-100"
          )}
          style={{
            transformOrigin: "12px 12px"
          }}
        />
        
        {/* Middle line */}
        <path
          d="M3 12h18"
          stroke={color}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          className={cn(
            "transition-all duration-300 ease-in-out",
            !isOpen ? "opacity-100 scale-x-100" : "opacity-0 scale-x-0"
          )}
        />
        
        {/* Bottom line */}
        <path
          d="M3 18h18"
          stroke={color}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          className={cn(
            "transition-all duration-300 ease-in-out origin-center",
            !isOpen 
              ? "rotate-0 translate-y-0 opacity-100" 
              : "-rotate-45 -translate-y-[6px] opacity-100"
          )}
          style={{
            transformOrigin: "12px 12px"
          }}
        />
      </svg>
    </div>
  );
}

interface ModernHamburgerProps {
  isOpen: boolean;
  className?: string;
  size?: number;
}

export function ModernHamburger({ isOpen, className, size = 20 }: ModernHamburgerProps) {
  return (
    <div className={cn("flex items-center justify-center", className)}>
      <div 
        className="relative flex flex-col justify-center items-center w-5 h-5 transition-all duration-300 ease-in-out"
        style={{ width: size, height: size }}
      >
        {/* Top bar */}
        <div
          className={cn(
            "absolute w-full h-0.5 bg-current rounded-full transition-all duration-300 ease-in-out",
            !isOpen 
              ? "rotate-0 -translate-y-1.5" 
              : "rotate-45 translate-y-0"
          )}
        />
        
        {/* Middle bar */}
        <div
          className={cn(
            "absolute w-full h-0.5 bg-current rounded-full transition-all duration-300 ease-in-out",
            !isOpen ? "opacity-100 scale-x-100" : "opacity-0 scale-x-0"
          )}
        />
        
        {/* Bottom bar */}
        <div
          className={cn(
            "absolute w-full h-0.5 bg-current rounded-full transition-all duration-300 ease-in-out",
            !isOpen 
              ? "rotate-0 translate-y-1.5" 
              : "-rotate-45 translate-y-0"
          )}
        />
      </div>
    </div>
  );
}

interface PulsatingHamburgerProps {
  isOpen: boolean;
  className?: string;
  size?: number;
}

export function PulsatingHamburger({ isOpen, className, size = 20 }: PulsatingHamburgerProps) {
  return (
    <div className={cn("flex items-center justify-center relative", className)}>
      {/* Pulsating background */}
      <div
        className={cn(
          "absolute rounded-full transition-all duration-300 ease-in-out",
          isOpen 
            ? "bg-primary/20 scale-110" 
            : "bg-transparent scale-100 hover:bg-accent/50 hover:scale-105"
        )}
        style={{
          width: size * 1.8,
          height: size * 1.8,
        }}
      />
      
      {/* Hamburger bars */}
      <div className="relative z-10">
        <ModernHamburger isOpen={isOpen} size={size} />
      </div>
    </div>
  );
}

interface SpringyHamburgerProps {
  isOpen: boolean;
  className?: string;
  size?: number;
}

export function SpringyHamburger({ isOpen, className, size = 20 }: SpringyHamburgerProps) {
  return (
    <div className={cn("flex items-center justify-center", className)}>
      <div 
        className="relative transition-all duration-500 ease-spring"
        style={{ 
          width: size, 
          height: size,
          transform: !isOpen ? "rotate(0deg)" : "rotate(180deg)"
        }}
      >
        {/* Top bar */}
        <div
          className={cn(
            "absolute w-full h-0.5 bg-current rounded-full transition-all duration-500",
            "ease-&lsqb;cubic-bezier(0.175,0.885,0.32,1.275)&rsqb;",
            !isOpen 
              ? "rotate-0 -translate-y-1.5 scale-x-100" 
              : "rotate-45 translate-y-0 scale-x-110"
          )}
          style={{
            top: "50%",
            left: "0",
            transformOrigin: "center"
          }}
        />
        
        {/* Middle bar */}
        <div
          className={cn(
            "absolute w-full h-0.5 bg-current rounded-full transition-all duration-300",
            "ease-&lsqb;cubic-bezier(0.175,0.885,0.32,1.275)&rsqb;",
            !isOpen 
              ? "opacity-100 scale-x-100 rotate-0" 
              : "opacity-0 scale-x-0 rotate-180"
          )}
          style={{
            top: "50%",
            left: "0",
            transformOrigin: "center"
          }}
        />
        
        {/* Bottom bar */}
        <div
          className={cn(
            "absolute w-full h-0.5 bg-current rounded-full transition-all duration-500",
            "ease-&lsqb;cubic-bezier(0.175,0.885,0.32,1.275)&rsqb;",
            !isOpen 
              ? "rotate-0 translate-y-1.5 scale-x-100" 
              : "-rotate-45 translate-y-0 scale-x-110"
          )}
          style={{
            top: "50%",
            left: "0",
            transformOrigin: "center"
          }}
        />
      </div>
    </div>
  );
}