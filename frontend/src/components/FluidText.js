import React, { useRef, useEffect, useState } from 'react';

const FluidText = ({ children, className = '', sensitivity = 1, ...props }) => {
  const textRef = useRef(null);
  const [mouseDistance, setMouseDistance] = useState(1);

  useEffect(() => {
    const handleMouseMove = (e) => {
      if (!textRef.current) return;

      const rect = textRef.current.getBoundingClientRect();
      const centerX = rect.left + rect.width / 2;
      const centerY = rect.top + rect.height / 2;
      
      const distance = Math.sqrt(
        Math.pow(e.clientX - centerX, 2) + Math.pow(e.clientY - centerY, 2)
      );
      
      // Maximum detection radius (in pixels)
      const maxRadius = 200;
      const normalizedDistance = Math.max(0, Math.min(1, distance / maxRadius));
      
      // Invert so closer mouse = higher value
      const proximity = 1 - normalizedDistance;
      
      setMouseDistance(proximity * sensitivity);
    };

    document.addEventListener('mousemove', handleMouseMove);
    
    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
    };
  }, [sensitivity]);

  const fluidStyle = {
    fontSize: `${1 + mouseDistance * 0.2}em`,
    fontWeight: 400 + (mouseDistance * 200),
    letterSpacing: `${mouseDistance * 0.5}px`,
    textShadow: mouseDistance > 0.3 ? `0 0 ${mouseDistance * 20}px rgba(59, 130, 246, ${mouseDistance * 0.3})` : 'none',
    transition: 'all 0.3s cubic-bezier(0.25, 0.46, 0.45, 0.94)',
  };

  return (
    <span
      ref={textRef}
      className={`fluid-text ${className}`}
      style={fluidStyle}
      {...props}
    >
      {children}
    </span>
  );
};

export default FluidText;