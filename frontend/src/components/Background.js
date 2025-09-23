import React, { useEffect, useRef, useState } from 'react';

const LuxuryBackground = ({ 
  currentView, 
  analysisData, 
  selectedColumn, 
  isLoading,
  // NEW: Status indicators
  autoSyncRunning = false,
  autoCreateRunning = false 
}) => {
  const canvasRef = useRef(null);
  const animationIdRef = useRef(null);
  const mouseRef = useRef({ x: 0, y: 0 });
  const dotsRef = useRef([]);
  const connectionsRef = useRef([]);
  const lastViewRef = useRef('');
  const lastColumnRef = useRef('');
  const [isTransitioning, setIsTransitioning] = useState(false);
  const [mouseZoom, setMouseZoom] = useState({ x: 0, y: 0, active: false });
  const animationTimeRef = useRef(0);

  // Enhanced configuration with status indicator support
  const config = {
    dotCount: 65,
    maxConnections: 2,
    connectionDistance: 200,
    cursorConnectionDistance: 180,
    dotSize: 2.2,
    lineWidth: 1.8,
    glowIntensity: 0.8,
    fadeSpeed: 0.04,
    transitionDuration: 400,
    zoomIntensity: 0.12,
    zoomRadius: 200,
    // Animation properties
    floatSpeed: 0.0008,
    floatAmplitude: 25,
    waveSpeed: 0.0015,
    breathingSpeed: 0.001,
    // NEW: Status indicator properties
    statusPulseSpeed: 0.002,
    statusGlowIntensity: 1.2,
    statusSizeMultiplier: 1.3
  };

  // Simple distance calculation utility
  const calculateDistance = (point1, point2) => {
    const dx = point1.x - point2.x;
    const dy = point1.y - point2.y;
    return Math.sqrt(dx * dx + dy * dy);
  };

  // Enhanced Dot class with status indicator support
  class Dot {
    constructor(x, y, index) {
      this.baseX = x;
      this.baseY = y;
      this.x = x;
      this.y = y;
      this.targetX = x;
      this.targetY = y;
      this.vx = 0;
      this.vy = 0;
      this.size = config.dotSize + Math.random() * 0.8;
      this.opacity = 0.5 + Math.random() * 0.3;
      this.baseOpacity = this.opacity;
      this.glowIntensity = 0;
      this.connections = [];
      this.lastConnectionTime = 0;
      
      // Animation properties
      this.floatOffsetX = Math.random() * Math.PI * 2;
      this.floatOffsetY = Math.random() * Math.PI * 2;
      this.floatSpeedMultiplier = 0.8 + Math.random() * 0.4;
      this.index = index;
      
      // Size breathing effect
      this.breathingOffset = Math.random() * Math.PI * 2;
      this.baseSize = this.size;
      
      // NEW: Status indicator properties
      this.zone = null;
      this.isStatusDot = false;
      this.statusType = null;
      this.statusPulseOffset = Math.random() * Math.PI * 2;
      this.statusTransition = 0; // 0 to 1 for smooth transitions
    }

    // NEW: Determine zone and update status
    updateZone(canvasWidth) {
      this.zone = this.baseX < canvasWidth / 2 ? 'left' : 'right';
      
      // Check if this dot should be a status indicator
      const wasStatusDot = this.isStatusDot;
      const oldStatusType = this.statusType;
      
      if (this.zone === 'left' && autoSyncRunning) {
        this.isStatusDot = true;
        this.statusType = 'sync';
      } else if (this.zone === 'right' && autoCreateRunning) {
        this.isStatusDot = true;
        this.statusType = 'create';
      } else {
        this.isStatusDot = false;
        this.statusType = null;
      }
      
      // Smooth transition for status change
      if (this.isStatusDot && (!wasStatusDot || oldStatusType !== this.statusType)) {
        // Becoming a status dot or changing type
        this.statusTransition = Math.min(1, this.statusTransition + 0.05);
      } else if (!this.isStatusDot && wasStatusDot) {
        // No longer a status dot
        this.statusTransition = Math.max(0, this.statusTransition - 0.03);
      } else if (this.isStatusDot) {
        // Maintain status dot state
        this.statusTransition = Math.min(1, this.statusTransition + 0.02);
      } else {
        // Normal dot
        this.statusTransition = Math.max(0, this.statusTransition - 0.02);
      }
    }

    update(time, canvasWidth) {
      // Update zone and status
      this.updateZone(canvasWidth);
      
      // Continuous floating animation
      const floatX = Math.sin(time * config.floatSpeed * this.floatSpeedMultiplier + this.floatOffsetX) * config.floatAmplitude;
      const floatY = Math.cos(time * config.floatSpeed * this.floatSpeedMultiplier + this.floatOffsetY) * config.floatAmplitude * 0.7;
      
      // Wave-like movement across the screen
      const waveX = Math.sin(time * config.waveSpeed + this.index * 0.1) * 15;
      const waveY = Math.cos(time * config.waveSpeed * 0.7 + this.index * 0.15) * 10;
      
      // NEW: Enhanced movement for status dots
      let statusFloatX = 0, statusFloatY = 0;
      if (this.isStatusDot && this.statusTransition > 0) {
        const statusPulse = Math.sin(time * config.statusPulseSpeed + this.statusPulseOffset);
        statusFloatX = statusPulse * 5 * this.statusTransition;
        statusFloatY = Math.cos(time * config.statusPulseSpeed * 0.7 + this.statusPulseOffset) * 3 * this.statusTransition;
      }
      
      // Update target position with all animations
      this.targetX = this.baseX + floatX + waveX + statusFloatX;
      this.targetY = this.baseY + floatY + waveY + statusFloatY;

      // Smooth movement to animated target position
      const dx = this.targetX - this.x;
      const dy = this.targetY - this.y;
      this.vx += dx * 0.08;
      this.vy += dy * 0.08;
      this.vx *= 0.90;
      this.vy *= 0.90;
      this.x += this.vx;
      this.y += this.vy;

      // Enhanced size calculation with status effects
      const breathingScale = 1 + Math.sin(time * config.breathingSpeed + this.breathingOffset) * 0.2;
      let statusSizeBoost = 1;
      
      if (this.isStatusDot && this.statusTransition > 0) {
        const statusPulse = Math.sin(time * config.statusPulseSpeed * 2 + this.statusPulseOffset);
        statusSizeBoost = 1 + (statusPulse * 0.3 + 0.3) * this.statusTransition * config.statusSizeMultiplier;
      }
      
      this.size = this.baseSize * breathingScale * statusSizeBoost;

      // Enhanced glow with status effects
      this.glowIntensity *= 0.92;
      
      if (this.isStatusDot && this.statusTransition > 0) {
        const statusGlow = Math.sin(time * config.statusPulseSpeed * 1.5 + this.statusPulseOffset) * 0.5 + 0.5;
        this.glowIntensity = Math.max(this.glowIntensity, statusGlow * config.statusGlowIntensity * this.statusTransition);
      }
      
      // Update opacity based on connections and status
      if (this.connections.length > 0) {
        this.opacity = Math.min(1, this.baseOpacity + 0.5);
      } else if (this.isStatusDot && this.statusTransition > 0) {
        this.opacity = Math.min(1, this.baseOpacity + 0.3 * this.statusTransition);
      } else {
        this.opacity = this.baseOpacity;
      }
    }

    updateBasePosition(newX, newY) {
      this.baseX = newX;
      this.baseY = newY;
    }

    draw(ctx) {
      const gradient = ctx.createRadialGradient(
        this.x, this.y, 0,
        this.x, this.y, this.size * 3
      );
      
      const alpha = this.opacity;
      const glowAlpha = Math.min(0.9, this.glowIntensity);
      
      // NEW: Status-aware colors
      if (this.isStatusDot && this.statusTransition > 0) {
        const normalAlpha = alpha * (1 - this.statusTransition);
        const statusAlpha = alpha * this.statusTransition;
        
        if (this.statusType === 'sync') {
          // Auto-sync colors (green/blue theme)
          gradient.addColorStop(0, `rgba(16, 185, 129, ${statusAlpha * 0.9})`);
          gradient.addColorStop(0.3, `rgba(34, 197, 94, ${statusAlpha * 0.7})`);
          gradient.addColorStop(0.7, `rgba(59, 130, 246, ${statusAlpha * 0.5})`);
          gradient.addColorStop(1, `rgba(148, 163, 184, ${normalAlpha * 0.3})`);
        } else if (this.statusType === 'create') {
          // Auto-create colors (blue/purple theme)
          gradient.addColorStop(0, `rgba(59, 130, 246, ${statusAlpha * 0.9})`);
          gradient.addColorStop(0.3, `rgba(99, 102, 241, ${statusAlpha * 0.7})`);
          gradient.addColorStop(0.7, `rgba(139, 92, 246, ${statusAlpha * 0.5})`);
          gradient.addColorStop(1, `rgba(148, 163, 184, ${normalAlpha * 0.3})`);
        }
        
        // Enhanced glow for status dots
        if (this.glowIntensity > 0.1) {
          const glowColor = this.statusType === 'sync' 
            ? `rgba(16, 185, 129, ${glowAlpha * this.statusTransition})`
            : `rgba(59, 130, 246, ${glowAlpha * this.statusTransition})`;
          ctx.shadowColor = glowColor;
          ctx.shadowBlur = 25 * this.statusTransition;
        }
      } else {
        // Normal dot colors
        gradient.addColorStop(0, `rgba(71, 85, 105, ${alpha})`);
        gradient.addColorStop(0.5, `rgba(100, 116, 139, ${alpha * 0.7})`);
        gradient.addColorStop(1, `rgba(148, 163, 184, 0)`);
        
        // Normal glow effect
        if (this.glowIntensity > 0.1) {
          ctx.shadowColor = `rgba(30, 64, 175, ${glowAlpha})`;
          ctx.shadowBlur = 18;
        }
      }

      ctx.fillStyle = gradient;
      ctx.beginPath();
      ctx.arc(this.x, this.y, this.size, 0, Math.PI * 2);
      ctx.fill();

      // Reset shadow
      ctx.shadowBlur = 0;
    }

    addGlow() {
      this.glowIntensity = 1;
      this.lastConnectionTime = Date.now();
    }
  }

  // Enhanced Connection class with status networking support
  class Connection {
    constructor(point1, point2, isCursorConnection = false) {
      this.point1 = point1;
      this.point2 = point2;
      this.isCursorConnection = isCursorConnection;
      this.opacity = 0;
      this.targetOpacity = isCursorConnection ? 0.9 : 0.7;
      this.createdAt = Date.now();
      this.isActive = true;
      
      // NEW: Status connection properties
      this.isStatusConnection = false;
      this.statusType = null;
      this.isLongDistance = false;
      this.pulseOffset = Math.random() * Math.PI * 2;
    }

    update() {
      if (this.isActive) {
        // Enhanced opacity for status connections
        const targetOpacity = this.isStatusConnection ? 0.9 : 
                            (this.isCursorConnection ? 0.9 : 0.7);
        this.opacity = Math.min(targetOpacity, this.opacity + 0.08);
      } else {
        this.opacity *= 0.88;
      }
      
      return this.opacity > 0.01;
    }

    draw(ctx) {
      if (this.opacity <= 0) return;

      const distance = calculateDistance(this.point1, this.point2);
      const maxDistance = this.isCursorConnection ? 
        config.cursorConnectionDistance : 
        (this.isLongDistance ? config.connectionDistance * 2 : config.connectionDistance);
      
      const distanceOpacity = 1 - (distance / maxDistance);
      let finalOpacity = this.opacity * distanceOpacity;

      if (finalOpacity <= 0) return;

      // NEW: Enhanced pulsing effect for status connections
      if (this.isStatusConnection) {
        const pulseEffect = Math.sin(animationTimeRef.current * 0.003 + this.pulseOffset) * 0.3 + 0.7;
        finalOpacity *= pulseEffect;
      }

      // Status-aware connection colors with enhanced effects
      const gradient = ctx.createLinearGradient(
        this.point1.x, this.point1.y,
        this.point2.x, this.point2.y
      );

      // Determine connection color scheme
      let statusType = this.statusType;
      if (!statusType) {
        if (this.point1.isStatusDot) statusType = this.point1.statusType;
        else if (this.point2.isStatusDot) statusType = this.point2.statusType;
      }

      if (this.isCursorConnection) {
        if (statusType === 'sync') {
          gradient.addColorStop(0, `rgba(16, 185, 129, ${finalOpacity})`);
          gradient.addColorStop(0.5, `rgba(34, 197, 94, ${finalOpacity * 1.1})`);
          gradient.addColorStop(1, `rgba(16, 185, 129, ${finalOpacity * 0.8})`);
        } else if (statusType === 'create') {
          gradient.addColorStop(0, `rgba(59, 130, 246, ${finalOpacity})`);
          gradient.addColorStop(0.5, `rgba(99, 102, 241, ${finalOpacity * 1.1})`);
          gradient.addColorStop(1, `rgba(59, 130, 246, ${finalOpacity * 0.8})`);
        } else {
          gradient.addColorStop(0, `rgba(30, 64, 175, ${finalOpacity})`);
          gradient.addColorStop(1, `rgba(15, 23, 42, ${finalOpacity * 0.8})`);
        }
      } else if (this.isStatusConnection) {
        // Enhanced status connection colors with pulsing gradient
        if (statusType === 'sync') {
          const pulseIntensity = Math.sin(animationTimeRef.current * 0.004 + this.pulseOffset) * 0.2 + 0.8;
          gradient.addColorStop(0, `rgba(16, 185, 129, ${finalOpacity * pulseIntensity})`);
          gradient.addColorStop(0.3, `rgba(34, 197, 94, ${finalOpacity * 1.2 * pulseIntensity})`);
          gradient.addColorStop(0.7, `rgba(59, 130, 246, ${finalOpacity * 0.8 * pulseIntensity})`);
          gradient.addColorStop(1, `rgba(16, 185, 129, ${finalOpacity * pulseIntensity})`);
        } else if (statusType === 'create') {
          const pulseIntensity = Math.sin(animationTimeRef.current * 0.004 + this.pulseOffset + Math.PI) * 0.2 + 0.8;
          gradient.addColorStop(0, `rgba(59, 130, 246, ${finalOpacity * pulseIntensity})`);
          gradient.addColorStop(0.3, `rgba(99, 102, 241, ${finalOpacity * 1.2 * pulseIntensity})`);
          gradient.addColorStop(0.7, `rgba(139, 92, 246, ${finalOpacity * 0.8 * pulseIntensity})`);
          gradient.addColorStop(1, `rgba(59, 130, 246, ${finalOpacity * pulseIntensity})`);
        }
      } else {
        // Regular connection colors
        if (statusType === 'sync') {
          gradient.addColorStop(0, `rgba(16, 185, 129, ${finalOpacity})`);
          gradient.addColorStop(0.5, `rgba(34, 197, 94, ${finalOpacity * 0.9})`);
          gradient.addColorStop(1, `rgba(16, 185, 129, ${finalOpacity})`);
        } else if (statusType === 'create') {
          gradient.addColorStop(0, `rgba(59, 130, 246, ${finalOpacity})`);
          gradient.addColorStop(0.5, `rgba(99, 102, 241, ${finalOpacity * 0.9})`);
          gradient.addColorStop(1, `rgba(59, 130, 246, ${finalOpacity})`);
        } else {
          gradient.addColorStop(0, `rgba(71, 85, 105, ${finalOpacity})`);
          gradient.addColorStop(0.5, `rgba(100, 116, 139, ${finalOpacity * 0.9})`);
          gradient.addColorStop(1, `rgba(71, 85, 105, ${finalOpacity})`);
        }
      }

      // Enhanced glow effects for status connections
      let shadowColor, shadowBlur;
      if (this.isStatusConnection) {
        const glowIntensity = Math.sin(animationTimeRef.current * 0.005 + this.pulseOffset) * 0.4 + 0.6;
        if (statusType === 'sync') {
          shadowColor = `rgba(16, 185, 129, ${finalOpacity * 0.8 * glowIntensity})`;
          shadowBlur = this.isLongDistance ? 20 : 15;
        } else if (statusType === 'create') {
          shadowColor = `rgba(59, 130, 246, ${finalOpacity * 0.8 * glowIntensity})`;
          shadowBlur = this.isLongDistance ? 20 : 15;
        }
      } else if (statusType === 'sync') {
        shadowColor = `rgba(16, 185, 129, ${finalOpacity * 0.6})`;
        shadowBlur = this.isCursorConnection ? 15 : 8;
      } else if (statusType === 'create') {
        shadowColor = `rgba(59, 130, 246, ${finalOpacity * 0.6})`;
        shadowBlur = this.isCursorConnection ? 15 : 8;
      } else {
        shadowColor = this.isCursorConnection ? 
          `rgba(30, 64, 175, ${finalOpacity * 0.6})` : 
          `rgba(71, 85, 105, ${finalOpacity * 0.4})`;
        shadowBlur = this.isCursorConnection ? 10 : 6;
      }

      // Enhanced line width for status connections
      const lineWidth = this.isStatusConnection ? config.lineWidth * 1.4 : config.lineWidth;

      ctx.shadowColor = shadowColor;
      ctx.shadowBlur = shadowBlur;
      ctx.strokeStyle = gradient;
      ctx.lineWidth = lineWidth;
      ctx.beginPath();
      ctx.moveTo(this.point1.x, this.point1.y);
      ctx.lineTo(this.point2.x, this.point2.y);
      ctx.stroke();

      // NEW: Add particle effect along status connections
      if (this.isStatusConnection && finalOpacity > 0.5) {
        this.drawConnectionParticle(ctx, finalOpacity);
      }

      // Reset shadow
      ctx.shadowBlur = 0;
    }

    // NEW: Draw moving particle along status connection
    drawConnectionParticle(ctx, opacity) {
      const time = animationTimeRef.current * 0.002;
      const progress = (Math.sin(time + this.pulseOffset) + 1) / 2; // 0 to 1
      
      const particleX = this.point1.x + (this.point2.x - this.point1.x) * progress;
      const particleY = this.point1.y + (this.point2.y - this.point1.y) * progress;
      
      const particleSize = 2;
      const particleOpacity = opacity * 0.8;
      
      const particleGradient = ctx.createRadialGradient(
        particleX, particleY, 0,
        particleX, particleY, particleSize * 2
      );
      
      if (this.statusType === 'sync') {
        particleGradient.addColorStop(0, `rgba(34, 197, 94, ${particleOpacity})`);
        particleGradient.addColorStop(1, `rgba(16, 185, 129, 0)`);
      } else if (this.statusType === 'create') {
        particleGradient.addColorStop(0, `rgba(99, 102, 241, ${particleOpacity})`);
        particleGradient.addColorStop(1, `rgba(59, 130, 246, 0)`);
      }
      
      ctx.shadowBlur = 8;
      ctx.shadowColor = this.statusType === 'sync' ? 
        `rgba(34, 197, 94, ${particleOpacity})` : 
        `rgba(99, 102, 241, ${particleOpacity})`;
      
      ctx.fillStyle = particleGradient;
      ctx.beginPath();
      ctx.arc(particleX, particleY, particleSize, 0, Math.PI * 2);
      ctx.fill();
      
      ctx.shadowBlur = 0;
    }

    deactivate() {
      this.isActive = false;
    }
  }

  // Initialize dots with better spacing and movement setup
  const initializeDots = (width, height) => {
    const dots = [];
    const minDistance = 140;
    
    for (let i = 0; i < config.dotCount; i++) {
      let x, y, validPosition = false, attempts = 0;
      
      while (!validPosition && attempts < 50) {
        x = 80 + Math.random() * (width - 160);
        y = 80 + Math.random() * (height - 160);
        
        validPosition = true;
        for (const existingDot of dots) {
          if (calculateDistance({ x, y }, { x: existingDot.baseX, y: existingDot.baseY }) < minDistance) {
            validPosition = false;
            break;
          }
        }
        attempts++;
      }
      
      if (validPosition) {
        dots.push(new Dot(x, y, i));
      }
    }
    return dots;
  };

  // Enhanced rearrange function that updates base positions
  const rearrangeDots = (width, height) => {
    const minDistance = 140;
    
    dotsRef.current.forEach((dot, index) => {
      let newX, newY, validPosition = false, attempts = 0;
      
      while (!validPosition && attempts < 30) {
        newX = 80 + Math.random() * (width - 160);
        newY = 80 + Math.random() * (height - 160);
        
        validPosition = true;
        for (let i = 0; i < dotsRef.current.length; i++) {
          if (i !== index) {
            const otherDot = dotsRef.current[i];
            if (calculateDistance({ x: newX, y: newY }, { x: otherDot.baseX, y: otherDot.baseY }) < minDistance) {
              validPosition = false;
              break;
            }
          }
        }
        attempts++;
      }
      
      if (validPosition) {
        dot.updateBasePosition(newX, newY);
      }
    });
  };

  // Find nearby dots for connections
  const findNearbyDots = (dot, excludeDots = []) => {
    return dotsRef.current
      .filter(other => other !== dot && !excludeDots.includes(other))
      .filter(other => calculateDistance(dot, other) <= config.connectionDistance)
      .sort((a, b) => calculateDistance(dot, a) - calculateDistance(dot, b));
  };

  // Update connections between dots
  const updateDotConnections = () => {
    // Clear existing non-cursor connections
    connectionsRef.current = connectionsRef.current.filter(conn => conn.isCursorConnection);

    // Create new dot-to-dot connections
    dotsRef.current.forEach(dot => {
      if (dot.connections.length < 1) {
        const nearby = findNearbyDots(dot, dot.connections);
        
        if (nearby.length > 0) {
          const nearestDot = nearby[0];
          
          const existingConnection = connectionsRef.current.find(conn => 
            (conn.point1 === dot && conn.point2 === nearestDot) ||
            (conn.point1 === nearestDot && conn.point2 === dot)
          );

          if (!existingConnection && nearestDot.connections.length < 1) {
            const connection = new Connection(dot, nearestDot);
            connectionsRef.current.push(connection);
            dot.connections.push(nearestDot);
            nearestDot.connections.push(dot);
            
            dot.addGlow();
            nearestDot.addGlow();
          }
        }
      }
    });
  };

  // Enhanced cursor connections with improved zoom effect
  const updateCursorConnections = () => {
    const mouse = mouseRef.current;
    
    // Remove old cursor connections
    connectionsRef.current = connectionsRef.current.filter(conn => {
      if (conn.isCursorConnection) {
        conn.deactivate();
        return conn.update();
      }
      return true;
    });

    // Find nearest dots to cursor
    const nearestDots = dotsRef.current
      .filter(dot => calculateDistance(dot, mouse) <= config.cursorConnectionDistance)
      .sort((a, b) => calculateDistance(a, mouse) - calculateDistance(b, mouse))
      .slice(0, config.maxConnections);

    // Update zoom effect with smoother activation
    if (nearestDots.length > 0) {
      const closestDistance = calculateDistance(nearestDots[0], mouse);
      const zoomStrength = Math.max(0, 1 - (closestDistance / config.cursorConnectionDistance));
      setMouseZoom({ 
        x: mouse.x, 
        y: mouse.y, 
        active: true,
        strength: zoomStrength
      });
    } else {
      setMouseZoom(prev => ({ ...prev, active: false, strength: 0 }));
    }

    // Create cursor connections
    nearestDots.forEach(dot => {
      const cursorPoint = { x: mouse.x, y: mouse.y };
      const connection = new Connection(cursorPoint, dot, true);
      connectionsRef.current.push(connection);
      dot.addGlow();
    });

    // Create triangular connections between cursor-connected dots
    if (nearestDots.length >= 2) {
      for (let i = 0; i < nearestDots.length - 1; i++) {
        for (let j = i + 1; j < nearestDots.length; j++) {
          const dot1 = nearestDots[i];
          const dot2 = nearestDots[j];
          
          if (calculateDistance(dot1, dot2) <= config.connectionDistance) {
            const connection = new Connection(dot1, dot2);
            connectionsRef.current.push(connection);
            dot1.addGlow();
            dot2.addGlow();
          }
        }
      }
    }
  };

  // Enhanced animation loop with time-based animation and status support
  const animate = () => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    const width = canvas.width;
    const height = canvas.height;
    
    // Update animation time
    animationTimeRef.current += 16;

    // Clear canvas with darker gradient background
    const gradient = ctx.createLinearGradient(0, 0, width, height);
    gradient.addColorStop(0, '#f8fafc');
    gradient.addColorStop(0.3, '#e2e8f0');
    gradient.addColorStop(0.7, '#cbd5e1');
    gradient.addColorStop(1, '#94a3b8');
    
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, width, height);

    // Update dots with time-based animation and status support
    dotsRef.current.forEach(dot => {
      dot.connections = [];
      dot.update(animationTimeRef.current, width);
    });

    // Update connections
    updateDotConnections();
    updateCursorConnections();

    // Clean up dead connections
    connectionsRef.current = connectionsRef.current.filter(conn => conn.update());

    // Draw connections
    connectionsRef.current.forEach(conn => conn.draw(ctx));

    // Draw dots
    dotsRef.current.forEach(dot => dot.draw(ctx));

    animationIdRef.current = requestAnimationFrame(animate);
  };

  // Enhanced mouse movement handler
  const handleMouseMove = (event) => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const rect = canvas.getBoundingClientRect();
    mouseRef.current = {
      x: event.clientX - rect.left,
      y: event.clientY - rect.top
    };
  };

  // Handle window resize
  const handleResize = () => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;

    if (dotsRef.current.length === 0) {
      dotsRef.current = initializeDots(canvas.width, canvas.height);
    } else {
      rearrangeDots(canvas.width, canvas.height);
    }
  };

  // Enhanced navigation change handler
  const handleNavigationChange = () => {
    if (currentView !== lastViewRef.current || selectedColumn !== lastColumnRef.current) {
      lastViewRef.current = currentView;
      lastColumnRef.current = selectedColumn;

      setIsTransitioning(true);

      // Faster fade out all connections
      connectionsRef.current.forEach(conn => conn.deactivate());

      // Much faster rearrangement
      setTimeout(() => {
        if (canvasRef.current) {
          rearrangeDots(canvasRef.current.width, canvasRef.current.height);
        }
        setIsTransitioning(false);
      }, 150);
    }
  };

  // Initialize and cleanup
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;

    dotsRef.current = initializeDots(canvas.width, canvas.height);
    animationTimeRef.current = 0;
    animate();

    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('resize', handleResize);

    return () => {
      if (animationIdRef.current) {
        cancelAnimationFrame(animationIdRef.current);
      }
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  // Handle navigation changes
  useEffect(() => {
    handleNavigationChange();
  }, [currentView, selectedColumn]);

  return (
    <>
      <canvas
        ref={canvasRef}
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          width: '100vw',
          height: '100vh',
          zIndex: -2,
          pointerEvents: 'none',
          backgroundColor: 'transparent'
        }}
      />
      
      {/* Enhanced transition indicator */}
      {isTransitioning && (
        <div
          style={{
            position: 'fixed',
            top: '20px',
            right: '20px',
            width: '10px',
            height: '10px',
            borderRadius: '50%',
            background: 'linear-gradient(135deg, #1e40af, #0f172a)',
            opacity: 0.8,
            zIndex: 5,
            animation: 'fastPulse 0.6s ease-in-out infinite'
          }}
        />
      )}
      
      {/* NEW: Status indicators */}
      {(autoSyncRunning || autoCreateRunning) && (
        <>
          {/* Auto-sync indicator (left) */}
          {autoSyncRunning && (
            <div
              style={{
                position: 'fixed',
                top: '20px',
                left: '20px',
                padding: '8px 16px',
                background: 'rgba(16, 185, 129, 0.15)',
                backdropFilter: 'blur(20px)',
                border: '1px solid rgba(16, 185, 129, 0.3)',
                borderRadius: '20px',
                color: '#059669',
                fontSize: '12px',
                fontWeight: '600',
                zIndex: 5,
                animation: 'statusPulse 2s ease-in-out infinite'
              }}
            >
              ● Auto-Sync Active
            </div>
          )}
          
          {/* Auto-create indicator (right) */}
          {autoCreateRunning && (
            <div
              style={{
                position: 'fixed',
                top: '20px',
                right: autoSyncRunning ? '180px' : '20px',
                padding: '8px 16px',
                background: 'rgba(59, 130, 246, 0.15)',
                backdropFilter: 'blur(20px)',
                border: '1px solid rgba(59, 130, 246, 0.3)',
                borderRadius: '20px',
                color: '#2563eb',
                fontSize: '12px',
                fontWeight: '600',
                zIndex: 5,
                animation: 'statusPulse 2s ease-in-out infinite 0.5s'
              }}
            >
              ● Auto-Create Active
            </div>
          )}
        </>
      )}
      
      <style jsx>{`
        @keyframes fastPulse {
          0%, 100% { 
            transform: scale(1);
            opacity: 0.8;
          }
          50% { 
            transform: scale(1.4);
            opacity: 1;
          }
        }
        
        @keyframes statusPulse {
          0%, 100% { 
            opacity: 0.8;
            transform: scale(1);
          }
          50% { 
            opacity: 1;
            transform: scale(1.05);
          }
        }
        
        @keyframes simpleRotate {
          from { transform: translate(-50%, -50%) rotate(45deg); }
          to { transform: translate(-50%, -50%) rotate(405deg); }
        }
      `}</style>
    </>
  );
};

export default LuxuryBackground;