import React, { useEffect, useRef, useState } from 'react';

const LuxuryBackground = ({ 
  currentView, 
  selectedColumn, 
  // Status indicators
  autoSyncRunning = false,
  autoCreateRunning = false 
}) => {
  const canvasRef = useRef(null);
  const animationIdRef = useRef(null);
  const mouseRef = useRef({ x: 0, y: 0, normalizedX: 0, normalizedY: 0 });
  const dotsRef = useRef([]);
  const connectionsRef = useRef([]);
  const lastViewRef = useRef('');
  const lastColumnRef = useRef('');
  const [isTransitioning, setIsTransitioning] = useState(false);
  const animationTimeRef = useRef(0);

  // Enhanced configuration with mouse clustering
  const config = {
    dotCount: 250,
    maxConnections: 2,
    connectionDistance: 200,
    cursorConnectionDistance: 210,
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
    // Status indicator properties
    statusPulseSpeed: 0.002,
    statusGlowIntensity: 1.2,
    statusSizeMultiplier: 1.3,
    // Mouse clustering properties - ENHANCED
    clusterRadius: 100,           // Larger area to gather dots from
    clusterStrength: 0.01,         // Stronger attraction
    maxClusterDistance: 5,      // How far dots can move to cluster
    clusterResponseSpeed: 0.15,   // How quickly dots respond to clustering
    minClusterDots: 6,            // Target number of dots to gather around mouse
    clusterZoneRadius: 120        // Tight cluster zone around mouse
  };

  // Simple distance calculation utility
  const calculateDistance = (point1, point2) => {
    const dx = point1.x - point2.x;
    const dy = point1.y - point2.y;
    return Math.sqrt(dx * dx + dy * dy);
  };

  // Enhanced Dot class with mouse attraction
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
      
      // Status indicator properties
      this.zone = null;
      this.isStatusDot = false;
      this.statusType = null;
      this.statusPulseOffset = Math.random() * Math.PI * 2;
      this.statusTransition = 0;
    }

    // Determine zone and update status
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
        this.statusTransition = Math.min(1, this.statusTransition + 0.05);
      } else if (!this.isStatusDot && wasStatusDot) {
        this.statusTransition = Math.max(0, this.statusTransition - 0.03);
      } else if (this.isStatusDot) {
        this.statusTransition = Math.min(1, this.statusTransition + 0.02);
      } else {
        this.statusTransition = Math.max(0, this.statusTransition - 0.02);
      }
    }

    // Enhanced update method with mouse clustering
    update(time, canvasWidth, mouse, allDots) {
      // Update zone and status
      this.updateZone(canvasWidth);
      
      // Ensure mouse object exists
      const safeMouseX = mouse?.x || 0;
      const safeMouseY = mouse?.y || 0;
      
      // Calculate distance to mouse
      const mouseDistance = Math.sqrt(
        Math.pow(safeMouseX - this.baseX, 2) + Math.pow(safeMouseY - this.baseY, 2)
      );
      
      // ENHANCED CLUSTERING LOGIC
      let clusterAttractionX = 0;
      let clusterAttractionY = 0;
      
      if (mouseDistance < config.clusterRadius) {
        // Find how many dots are already in the cluster zone
        const dotsInClusterZone = allDots.filter(dot => {
          const distToMouse = Math.sqrt(
            Math.pow(safeMouseX - dot.x, 2) + Math.pow(safeMouseY - dot.y, 2)
          );
          return distToMouse < config.clusterZoneRadius;
        }).length;
        
        // If we need more dots in the cluster, attract this dot more strongly
        const clusterNeedFactor = Math.max(0, (config.minClusterDots - dotsInClusterZone) / config.minClusterDots);
        
        // Base attraction strength
        let attractionFactor = (1 - mouseDistance / config.clusterRadius) * config.clusterStrength;
        
        // Boost attraction if we need more dots in the cluster
        attractionFactor *= (1 + clusterNeedFactor * 2);
        
        // Make dots from farther away come to join the cluster
        if (mouseDistance > config.clusterZoneRadius && clusterNeedFactor > 0.3) {
          attractionFactor *= 1.5; // Extra boost for distant dots
        }
        
        // Calculate direction vector toward mouse
        if (mouseDistance > 0) {
          const directionX = (safeMouseX - this.baseX) / mouseDistance;
          const directionY = (safeMouseY - this.baseY) / mouseDistance;
          
          // Apply clustering with enhanced distance
          const clusterDistance = Math.min(config.maxClusterDistance, mouseDistance * attractionFactor);
          clusterAttractionX = directionX * clusterDistance * attractionFactor;
          clusterAttractionY = directionY * clusterDistance * attractionFactor;
        }
      }
      
      // Continuous floating animation (heavily reduced when clustering)
      const floatReduction = mouseDistance < config.clusterRadius ? 0.1 : 1;
      const floatX = Math.sin(time * config.floatSpeed * this.floatSpeedMultiplier + this.floatOffsetX) * config.floatAmplitude * floatReduction;
      const floatY = Math.cos(time * config.floatSpeed * this.floatSpeedMultiplier + this.floatOffsetY) * config.floatAmplitude * 0.7 * floatReduction;
      
      // Wave-like movement (also heavily reduced when clustering)
      const waveX = Math.sin(time * config.waveSpeed + this.index * 0.1) * 15 * floatReduction;
      const waveY = Math.cos(time * config.waveSpeed * 0.7 + this.index * 0.15) * 10 * floatReduction;
      
      // Status-based movement
      let statusFloatX = 0, statusFloatY = 0;
      if (this.isStatusDot && this.statusTransition > 0) {
        const statusPulse = Math.sin(time * config.statusPulseSpeed + this.statusPulseOffset);
        statusFloatX = statusPulse * 5 * this.statusTransition;
        statusFloatY = Math.cos(time * config.statusPulseSpeed * 0.7 + this.statusPulseOffset) * 3 * this.statusTransition;
      }
      
      // Combine all movements - clustering dominates when mouse is near
      this.targetX = this.baseX + floatX + waveX + clusterAttractionX + statusFloatX;
      this.targetY = this.baseY + floatY + waveY + clusterAttractionY + statusFloatY;

      // Enhanced smooth movement for clustering effect
      const dx = this.targetX - this.x;
      const dy = this.targetY - this.y;
      
      // Faster response when clustering
      const responseSpeed = mouseDistance < config.clusterRadius ? config.clusterResponseSpeed : 0.08;
      this.vx += dx * responseSpeed;
      this.vy += dy * responseSpeed;
      this.vx *= 0.85; // Slightly more damping for smoother clustering
      this.vy *= 0.85;
      this.x += this.vx;
      this.y += this.vy;

      // Enhanced size calculation for clustered dots
      const breathingScale = 1 + Math.sin(time * config.breathingSpeed + this.breathingOffset) * 0.2;
      let statusSizeBoost = 1;
      
      // Dots in cluster get larger and more prominent
      const clusterSizeBoost = mouseDistance < config.clusterZoneRadius ? 1.4 : 
                              mouseDistance < config.clusterRadius ? 1.2 : 1;
      
      if (this.isStatusDot && this.statusTransition > 0) {
        const statusPulse = Math.sin(time * config.statusPulseSpeed * 2 + this.statusPulseOffset);
        statusSizeBoost = 1 + (statusPulse * 0.3 + 0.3) * this.statusTransition * config.statusSizeMultiplier;
      }
      
      this.size = this.baseSize * breathingScale * statusSizeBoost * clusterSizeBoost;

      // Enhanced glow effects for clustering
      this.glowIntensity *= 0.88;
      
      // Strong glow for clustered dots
      if (mouseDistance < config.clusterZoneRadius) {
        const clusterGlow = 0.9;
        this.glowIntensity = Math.max(this.glowIntensity, clusterGlow);
      } else if (mouseDistance < config.clusterRadius) {
        const clusterGlow = (1 - mouseDistance / config.clusterRadius) * 0.7;
        this.glowIntensity = Math.max(this.glowIntensity, clusterGlow);
      }
      
      if (this.isStatusDot && this.statusTransition > 0) {
        const statusGlow = Math.sin(time * config.statusPulseSpeed * 1.5 + this.statusPulseOffset) * 0.5 + 0.5;
        this.glowIntensity = Math.max(this.glowIntensity, statusGlow * config.statusGlowIntensity * this.statusTransition);
      }
      
      // Enhanced opacity for clustered dots
      if (this.connections.length > 0) {
        this.opacity = Math.min(1, this.baseOpacity + 0.5);
      } else if (this.isStatusDot && this.statusTransition > 0) {
        this.opacity = Math.min(1, this.baseOpacity + 0.3 * this.statusTransition);
      } else if (mouseDistance < config.clusterZoneRadius) {
        // Dots in tight cluster are very visible
        this.opacity = Math.min(1, this.baseOpacity + 0.6);
      } else if (mouseDistance < config.clusterRadius) {
        // Dots being attracted are more visible
        const clusterOpacityBoost = (1 - mouseDistance / config.clusterRadius) * 0.4;
        this.opacity = Math.min(1, this.baseOpacity + clusterOpacityBoost);
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
      
      // Status-aware colors
      if (this.isStatusDot && this.statusTransition > 0) {
        const normalAlpha = alpha * (1 - this.statusTransition);
        const statusAlpha = alpha * this.statusTransition;
        
        if (this.statusType === 'sync') {
          gradient.addColorStop(0, `rgba(16, 185, 129, ${statusAlpha * 0.9})`);
          gradient.addColorStop(0.3, `rgba(34, 197, 94, ${statusAlpha * 0.7})`);
          gradient.addColorStop(0.7, `rgba(59, 130, 246, ${statusAlpha * 0.5})`);
          gradient.addColorStop(1, `rgba(148, 163, 184, ${normalAlpha * 0.3})`);
        } else if (this.statusType === 'create') {
          gradient.addColorStop(0, `rgba(59, 130, 246, ${statusAlpha * 0.9})`);
          gradient.addColorStop(0.3, `rgba(99, 102, 241, ${statusAlpha * 0.7})`);
          gradient.addColorStop(0.7, `rgba(139, 92, 246, ${statusAlpha * 0.5})`);
          gradient.addColorStop(1, `rgba(148, 163, 184, ${normalAlpha * 0.3})`);
        }
        
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
        
        if (this.glowIntensity > 0.1) {
          ctx.shadowColor = `rgba(30, 64, 175, ${glowAlpha})`;
          ctx.shadowBlur = 18;
        }
      }

      ctx.fillStyle = gradient;
      ctx.beginPath();
      ctx.arc(this.x, this.y, this.size, 0, Math.PI * 2);
      ctx.fill();

      ctx.shadowBlur = 0;
    }

    addGlow() {
      this.glowIntensity = 1;
      this.lastConnectionTime = Date.now();
    }
  }

  // Connection class
  class Connection {
    constructor(point1, point2, isCursorConnection = false) {
      this.point1 = point1;
      this.point2 = point2;
      this.isCursorConnection = isCursorConnection;
      this.opacity = 0;
      this.targetOpacity = isCursorConnection ? 0.9 : 0.7;
      this.createdAt = Date.now();
      this.isActive = true;
      this.isStatusConnection = false;
      this.statusType = null;
      this.isLongDistance = false;
      this.pulseOffset = Math.random() * Math.PI * 2;
    }

    update() {
      if (this.isActive) {
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

      if (this.isStatusConnection) {
        const pulseEffect = Math.sin(animationTimeRef.current * 0.003 + this.pulseOffset) * 0.3 + 0.7;
        finalOpacity *= pulseEffect;
      }

      const gradient = ctx.createLinearGradient(
        this.point1.x, this.point1.y,
        this.point2.x, this.point2.y
      );

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
      } else {
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

      const lineWidth = this.isStatusConnection ? config.lineWidth * 1.4 : config.lineWidth;

      ctx.strokeStyle = gradient;
      ctx.lineWidth = lineWidth;
      ctx.beginPath();
      ctx.moveTo(this.point1.x, this.point1.y);
      ctx.lineTo(this.point2.x, this.point2.y);
      ctx.stroke();
    }

    deactivate() {
      this.isActive = false;
    }
  }

  // Initialize dots
  const initializeDots = (width, height) => {
    const dots = [];
    const minDistance = 120;
    
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

  // Rearrange dots
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

  // Find nearby dots
  const findNearbyDots = (dot, excludeDots = []) => {
    return dotsRef.current
      .filter(other => other !== dot && !excludeDots.includes(other))
      .filter(other => calculateDistance(dot, other) <= config.connectionDistance)
      .sort((a, b) => calculateDistance(dot, a) - calculateDistance(dot, b));
  };

  // Update dot connections
  const updateDotConnections = () => {
    connectionsRef.current = connectionsRef.current.filter(conn => conn.isCursorConnection);

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

  // Update cursor connections
  const updateCursorConnections = () => {
    const mouse = mouseRef.current;
    
    connectionsRef.current = connectionsRef.current.filter(conn => {
      if (conn.isCursorConnection) {
        conn.deactivate();
        return conn.update();
      }
      return true;
    });

    const nearestDots = dotsRef.current
      .filter(dot => calculateDistance(dot, mouse) <= config.cursorConnectionDistance)
      .sort((a, b) => calculateDistance(a, mouse) - calculateDistance(b, mouse))
      .slice(0, config.maxConnections);

    // Mouse zoom effect removed for performance

    nearestDots.forEach(dot => {
      const cursorPoint = { x: mouse.x, y: mouse.y };
      const connection = new Connection(cursorPoint, dot, true);
      connectionsRef.current.push(connection);
      dot.addGlow();
    });

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

  // Animation loop
  const animate = () => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    const width = canvas.width;
    const height = canvas.height;
    
    animationTimeRef.current += 16;

    // Clear canvas
    const gradient = ctx.createLinearGradient(0, 0, width, height);
    gradient.addColorStop(0, '#f8fafc');
    gradient.addColorStop(0.3, '#e2e8f0');
    gradient.addColorStop(0.7, '#cbd5e1');
    gradient.addColorStop(1, '#94a3b8');
    
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, width, height);

    // Update dots with mouse clustering - pass all dots for clustering logic
    dotsRef.current.forEach(dot => {
      dot.connections = [];
      dot.update(animationTimeRef.current, width, mouseRef.current, dotsRef.current);
    });

    updateDotConnections();
    updateCursorConnections();

    connectionsRef.current = connectionsRef.current.filter(conn => conn.update());

    connectionsRef.current.forEach(conn => conn.draw(ctx));
    dotsRef.current.forEach(dot => dot.draw(ctx));

    animationIdRef.current = requestAnimationFrame(animate);
  };

  // Mouse movement handler
  const handleMouseMove = (event) => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const rect = canvas.getBoundingClientRect();
    mouseRef.current = {
      x: event.clientX - rect.left,
      y: event.clientY - rect.top,
      normalizedX: (event.clientX / window.innerWidth) * 2 - 1,
      normalizedY: -(event.clientY / window.innerHeight) * 2 + 1
    };
  };

  // Handle resize
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

  // Handle navigation changes
  const handleNavigationChange = () => {
    if (currentView !== lastViewRef.current || selectedColumn !== lastColumnRef.current) {
      lastViewRef.current = currentView;
      lastColumnRef.current = selectedColumn;

      setIsTransitioning(true);

      connectionsRef.current.forEach(conn => conn.deactivate());

      setTimeout(() => {
        if (canvasRef.current) {
          rearrangeDots(canvasRef.current.width, canvasRef.current.height);
        }
        setIsTransitioning(false);
      }, 150);
    }
  };

  // Initialize
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Handle navigation changes
  useEffect(() => {
    handleNavigationChange();
    // eslint-disable-next-line react-hooks/exhaustive-deps
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
      `}</style>
    </>
  );
};

export default LuxuryBackground;