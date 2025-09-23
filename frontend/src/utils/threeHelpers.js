import * as THREE from 'three';

/**
 * Three.js utility functions for the dashboard background
 */

// Color schemes for different dashboard states
export const ColorSchemes = {
  default: {
    primary: 0x3b82f6,
    secondary: 0x06b6d4,
    accent: 0x8b5cf6,
    binary: 0x00aaff,
    success: 0x10b981,
    warning: 0xf59e0b,
    error: 0xef4444
  },
  analyze: {
    primary: 0x8b5cf6,
    secondary: 0x06b6d4,
    accent: 0x3b82f6,
    binary: 0xaa88ff,
    success: 0x10b981,
    warning: 0xf59e0b,
    error: 0xef4444
  },
  sync: {
    primary: 0x10b981,
    secondary: 0x06b6d4,
    accent: 0x3b82f6,
    binary: 0x88ffaa,
    success: 0x10b981,
    warning: 0xf59e0b,
    error: 0xef4444
  }
};

/**
 * Create smooth animation between two positions
 * @param {THREE.Object3D} object - The object to animate
 * @param {Object} targetPosition - Target position {x, y, z}
 * @param {number} duration - Animation duration in ms
 * @param {Function} onComplete - Callback when animation completes
 */
export const animateToPosition = (object, targetPosition, duration = 1000, onComplete = null) => {
  // Store original position if not already stored
  if (!object.userData.originalPosition) {
    object.userData.originalPosition = {
      x: object.position.x,
      y: object.position.y,
      z: object.position.z
    };
  }

  const startPosition = {
    x: object.position.x,
    y: object.position.y,
    z: object.position.z
  };
  
  const startTime = Date.now();
  
  const animateStep = () => {
    const elapsed = Date.now() - startTime;
    const progress = Math.min(elapsed / duration, 1);
    
    // Smooth easing function (ease-in-out)
    const easeInOut = progress < 0.5 
      ? 2 * progress * progress 
      : 1 - Math.pow(-2 * progress + 2, 3) / 2;
    
    object.position.x = startPosition.x + (targetPosition.x - startPosition.x) * easeInOut;
    object.position.y = startPosition.y + (targetPosition.y - startPosition.y) * easeInOut;
    object.position.z = startPosition.z + (targetPosition.z - startPosition.z) * easeInOut;
    
    if (progress < 1) {
      requestAnimationFrame(animateStep);
    } else if (onComplete) {
      onComplete();
    }
  };
  
  animateStep();
};

/**
 * Create smooth color transition
 * @param {THREE.Material} material - Material to animate
 * @param {number} targetColor - Target color as hex
 * @param {number} duration - Animation duration in ms
 */
export const animateToColor = (material, targetColor, duration = 1000) => {
  const startColor = material.color.getHex();
  const startTime = Date.now();
  
  const animateStep = () => {
    const elapsed = Date.now() - startTime;
    const progress = Math.min(elapsed / duration, 1);
    
    const easeInOut = progress < 0.5 
      ? 2 * progress * progress 
      : 1 - Math.pow(-2 * progress + 2, 3) / 2;
    
    // Interpolate between colors
    const currentColor = new THREE.Color(startColor).lerp(new THREE.Color(targetColor), easeInOut);
    material.color.copy(currentColor);
    
    if (progress < 1) {
      requestAnimationFrame(animateStep);
    }
  };
  
  animateStep();
};

/**
 * Create text formation positions for logos
 * @param {string} text - Text to create positions for
 * @param {number} scale - Scale of the text
 * @returns {Array} Array of positions for each character
 */
export const getTextFormationPositions = (text, scale = 1) => {
  const positions = [];
  const spacing = 3 * scale;
  const startX = -(text.length * spacing) / 2;
  
  for (let i = 0; i < text.length; i++) {
    if (text[i] === ' ') continue;
    positions.push({
      x: startX + (i * spacing),
      y: 0,
      z: 0,
      char: text[i]
    });
  }
  
  return positions;
};

/**
 * Create circular formation positions
 * @param {number} count - Number of elements
 * @param {number} radius - Radius of the circle
 * @param {number} heightVariation - Variation in Y position
 * @returns {Array} Array of positions
 */
export const getCircularFormationPositions = (count, radius = 10, heightVariation = 0) => {
  const positions = [];
  
  for (let i = 0; i < count; i++) {
    const angle = (i / count) * Math.PI * 2;
    positions.push({
      x: Math.cos(angle) * radius,
      y: (Math.random() - 0.5) * heightVariation,
      z: Math.sin(angle) * radius
    });
  }
  
  return positions;
};

/**
 * Create spiral formation positions
 * @param {number} count - Number of elements
 * @param {number} radius - Base radius
 * @param {number} height - Height of spiral
 * @returns {Array} Array of positions
 */
export const getSpiralFormationPositions = (count, radius = 8, height = 15) => {
  const positions = [];
  
  for (let i = 0; i < count; i++) {
    const t = i / count;
    const angle = t * Math.PI * 6; // 3 full rotations
    const currentRadius = radius * (1 - t * 0.5); // Spiral inward
    
    positions.push({
      x: Math.cos(angle) * currentRadius,
      y: (t - 0.5) * height,
      z: Math.sin(angle) * currentRadius
    });
  }
  
  return positions;
};

/**
 * Create sync logo formation (rotating arrows)
 * @param {number} count - Number of elements
 * @param {number} radius - Radius of formation
 * @returns {Array} Array of positions for sync symbol
 */
export const getSyncLogoPositions = (count, radius = 8) => {
  const positions = [];
  const angleStep = (Math.PI * 2) / count;
  
  for (let i = 0; i < count; i++) {
    const angle = i * angleStep;
    // Create arrow-like formation
    const distance = radius + Math.sin(angle * 2) * 2;
    
    positions.push({
      x: Math.cos(angle) * distance,
      y: Math.sin(angle * 3) * 2,
      z: Math.sin(angle) * distance,
      rotation: angle
    });
  }
  
  return positions;
};

/**
 * Create analyze logo formation (chart-like patterns)
 * @param {number} count - Number of elements
 * @returns {Array} Array of positions for analysis visualization
 */
export const getAnalyzeLogoPositions = (count) => {
  const positions = [];
  const barsCount = Math.min(count, 7); // Max 7 bars for "ANALYZE"
  const spacing = 4;
  const startX = -(barsCount * spacing) / 2;
  
  for (let i = 0; i < barsCount && i < count; i++) {
    const height = 2 + Math.sin(i * 0.8) * 3; // Varying heights like a chart
    positions.push({
      x: startX + (i * spacing),
      y: height / 2,
      z: 0,
      height: height
    });
  }
  
  // Add remaining elements in a grid pattern around the bars
  for (let i = barsCount; i < count; i++) {
    const gridIndex = i - barsCount;
    const gridSize = Math.ceil(Math.sqrt(count - barsCount));
    const row = Math.floor(gridIndex / gridSize);
    const col = gridIndex % gridSize;
    
    positions.push({
      x: -8 + (col * 4),
      y: 6 + (row * 3),
      z: -3,
      height: 1
    });
  }
  
  return positions;
};

/**
 * Create ticket status visualization positions
 * @param {Object} analysisData - Analysis data with ticket counts
 * @param {number} totalElements - Total number of elements to position
 * @returns {Array} Array of positions representing ticket statuses
 */
export const getTicketVisualizationPositions = (analysisData, totalElements) => {
  if (!analysisData || !analysisData.summary) {
    return [];
  }
  
  const positions = [];
  const { summary } = analysisData;
  const total = summary.matched + summary.mismatched + summary.missing_youtrack;
  
  if (total === 0) return positions;
  
  // Calculate proportions
  const matchedRatio = summary.matched / total;
  const mismatchedRatio = summary.mismatched / total;
  const missingRatio = summary.missing_youtrack / total;
  
  const matchedCount = Math.floor(totalElements * matchedRatio);
  const mismatchedCount = Math.floor(totalElements * mismatchedRatio);
  const missingCount = totalElements - matchedCount - mismatchedCount;
  
  // Create positions for matched tickets (left side, green)
  for (let i = 0; i < matchedCount; i++) {
    positions.push({
      x: -15 + (Math.random() - 0.5) * 8,
      y: Math.random() * 10,
      z: (Math.random() - 0.5) * 6,
      status: 'matched',
      color: ColorSchemes.default.success
    });
  }
  
  // Create positions for mismatched tickets (center, yellow)
  for (let i = 0; i < mismatchedCount; i++) {
    positions.push({
      x: (Math.random() - 0.5) * 8,
      y: Math.random() * 10,
      z: (Math.random() - 0.5) * 6,
      status: 'mismatched',
      color: ColorSchemes.default.warning
    });
  }
  
  // Create positions for missing tickets (right side, blue)
  for (let i = 0; i < missingCount; i++) {
    positions.push({
      x: 15 + (Math.random() - 0.5) * 8,
      y: Math.random() * 10,
      z: (Math.random() - 0.5) * 6,
      status: 'missing',
      color: ColorSchemes.default.primary
    });
  }
  
  return positions;
};

/**
 * Create pulsing animation effect
 * @param {THREE.Object3D} object - Object to animate
 * @param {number} speed - Animation speed
 * @param {number} intensity - Pulse intensity
 */
export const addPulseAnimation = (object, speed = 0.02, intensity = 0.3) => {
  object.userData.pulseSpeed = speed;
  object.userData.pulseIntensity = intensity;
  object.userData.originalScale = object.scale.clone();
};

/**
 * Update pulse animation for an object
 * @param {THREE.Object3D} object - Object to update
 * @param {number} time - Current time
 */
export const updatePulseAnimation = (object, time) => {
  if (object.userData.pulseSpeed && object.userData.originalScale) {
    const pulse = 1 + Math.sin(time * object.userData.pulseSpeed) * object.userData.pulseIntensity;
    object.scale.copy(object.userData.originalScale).multiplyScalar(pulse);
  }
};

/**
 * Create flowing animation along a curve
 * @param {THREE.Curve} curve - The curve to follow
 * @param {number} speed - Animation speed
 * @returns {Function} Update function for the animation
 */
export const createFlowAnimation = (curve, speed = 0.01) => {
  let progress = 0;
  
  return (particles) => {
    progress += speed;
    if (progress > 1) progress = 0;
    
    const positions = particles.geometry.attributes.position.array;
    const particleCount = positions.length / 3;
    
    for (let i = 0; i < particleCount; i++) {
      const t = (i / particleCount + progress) % 1;
      const point = curve.getPoint(t);
      
      positions[i * 3] = point.x;
      positions[i * 3 + 1] = point.y;
      positions[i * 3 + 2] = point.z;
    }
    
    particles.geometry.attributes.position.needsUpdate = true;
  };
};

/**
 * Create wave animation for a plane geometry
 * @param {THREE.PlaneGeometry} geometry - Plane geometry to animate
 * @param {number} speed - Animation speed
 * @param {number} amplitude - Wave amplitude
 * @returns {Function} Update function for the animation
 */
export const createWaveAnimation = (geometry, speed = 0.01, amplitude = 2) => {
  const originalPositions = [...geometry.attributes.position.array];
  
  return (time) => {
    const positions = geometry.attributes.position.array;
    
    for (let i = 0; i < positions.length; i += 3) {
      const x = originalPositions[i];
      const z = originalPositions[i + 2];
      
      positions[i + 1] = originalPositions[i + 1] + 
        Math.sin(x * 0.1 + time * speed) * 
        Math.cos(z * 0.1 + time * speed) * amplitude;
    }
    
    geometry.attributes.position.needsUpdate = true;
  };
};

/**
 * Create random floating animation
 * @param {THREE.Object3D} object - Object to animate
 * @param {number} range - Range of movement
 * @param {number} speed - Animation speed
 */
export const addFloatingAnimation = (object, range = 2, speed = 0.5) => {
  object.userData.floatingRange = range;
  object.userData.floatingSpeed = speed;
  object.userData.floatingOffset = Math.random() * Math.PI * 2;
  
  if (!object.userData.originalPosition) {
    object.userData.originalPosition = {
      x: object.position.x,
      y: object.position.y,
      z: object.position.z
    };
  }
};

/**
 * Update floating animation for an object
 * @param {THREE.Object3D} object - Object to update
 * @param {number} time - Current time
 */
export const updateFloatingAnimation = (object, time) => {
  if (object.userData.originalPosition && object.userData.floatingRange) {
    const { originalPosition, floatingRange, floatingSpeed, floatingOffset } = object.userData;
    
    object.position.y = originalPosition.y + 
      Math.sin(time * floatingSpeed + floatingOffset) * floatingRange;
  }
};

/**
 * Create rotation animation
 * @param {THREE.Object3D} object - Object to animate
 * @param {Object} rotationSpeed - Rotation speeds {x, y, z}
 */
export const addRotationAnimation = (object, rotationSpeed = { x: 0, y: 0.01, z: 0 }) => {
  object.userData.rotationSpeed = rotationSpeed;
};

/**
 * Update rotation animation for an object
 * @param {THREE.Object3D} object - Object to update
 */
export const updateRotationAnimation = (object) => {
  if (object.userData.rotationSpeed) {
    const { rotationSpeed } = object.userData;
    object.rotation.x += rotationSpeed.x;
    object.rotation.y += rotationSpeed.y;
    object.rotation.z += rotationSpeed.z;
  }
};

/**
 * Create opacity fade animation
 * @param {THREE.Material} material - Material to animate
 * @param {number} targetOpacity - Target opacity
 * @param {number} duration - Animation duration in ms
 */
export const animateOpacity = (material, targetOpacity, duration = 1000) => {
  const startOpacity = material.opacity;
  const startTime = Date.now();
  
  const animateStep = () => {
    const elapsed = Date.now() - startTime;
    const progress = Math.min(elapsed / duration, 1);
    
    const easeInOut = progress < 0.5 
      ? 2 * progress * progress 
      : 1 - Math.pow(-2 * progress + 2, 3) / 2;
    
    material.opacity = startOpacity + (targetOpacity - startOpacity) * easeInOut;
    
    if (progress < 1) {
      requestAnimationFrame(animateStep);
    }
  };
  
  animateStep();
};

/**
 * Create scale animation
 * @param {THREE.Object3D} object - Object to animate
 * @param {number} targetScale - Target scale
 * @param {number} duration - Animation duration in ms
 */
export const animateScale = (object, targetScale, duration = 1000) => {
  const startScale = object.scale.x; // Assuming uniform scale
  const startTime = Date.now();
  
  const animateStep = () => {
    const elapsed = Date.now() - startTime;
    const progress = Math.min(elapsed / duration, 1);
    
    const easeInOut = progress < 0.5 
      ? 2 * progress * progress 
      : 1 - Math.pow(-2 * progress + 2, 3) / 2;
    
    const currentScale = startScale + (targetScale - startScale) * easeInOut;
    object.scale.setScalar(currentScale);
    
    if (progress < 1) {
      requestAnimationFrame(animateStep);
    }
  };
  
  animateStep();
};

/**
 * Dispose of Three.js objects properly to prevent memory leaks
 * @param {THREE.Object3D} object - Object to dispose
 */
export const disposeObject = (object) => {
  if (object.geometry) {
    object.geometry.dispose();
  }
  
  if (object.material) {
    if (Array.isArray(object.material)) {
      object.material.forEach(material => material.dispose());
    } else {
      object.material.dispose();
    }
  }
  
  if (object.texture) {
    object.texture.dispose();
  }
  
  // Recursively dispose children
  while (object.children.length > 0) {
    const child = object.children[0];
    object.remove(child);
    disposeObject(child);
  }
};

/**
 * Create smooth camera movement
 * @param {THREE.Camera} camera - Camera to animate
 * @param {Object} targetPosition - Target position {x, y, z}
 * @param {Object} targetLookAt - Target look at point {x, y, z}
 * @param {number} duration - Animation duration in ms
 */
export const animateCamera = (camera, targetPosition, targetLookAt = { x: 0, y: 0, z: 0 }, duration = 2000) => {
  const startPosition = {
    x: camera.position.x,
    y: camera.position.y,
    z: camera.position.z
  };
  
  const startTime = Date.now();
  
  const animateStep = () => {
    const elapsed = Date.now() - startTime;
    const progress = Math.min(elapsed / duration, 1);
    
    // Smooth easing for camera movement
    const easeInOut = progress < 0.5 
      ? 2 * progress * progress 
      : 1 - Math.pow(-2 * progress + 2, 3) / 2;
    
    camera.position.x = startPosition.x + (targetPosition.x - startPosition.x) * easeInOut;
    camera.position.y = startPosition.y + (targetPosition.y - startPosition.y) * easeInOut;
    camera.position.z = startPosition.z + (targetPosition.z - startPosition.z) * easeInOut;
    
    camera.lookAt(targetLookAt.x, targetLookAt.y, targetLookAt.z);
    
    if (progress < 1) {
      requestAnimationFrame(animateStep);
    }
  };
  
  animateStep();
};

/**
 * Create particle explosion effect
 * @param {THREE.Vector3} position - Explosion center
 * @param {number} particleCount - Number of particles
 * @param {number} color - Particle color
 * @param {THREE.Scene} scene - Scene to add particles to
 * @returns {THREE.Points} Particle system
 */
export const createExplosion = (position, particleCount = 50, color = 0xffffff, scene) => {
  const particles = new THREE.BufferGeometry();
  const positions = [];
  const velocities = [];
  
  for (let i = 0; i < particleCount; i++) {
    // Starting position
    positions.push(position.x, position.y, position.z);
    
    // Random velocity
    velocities.push(
      (Math.random() - 0.5) * 20,
      (Math.random() - 0.5) * 20,
      (Math.random() - 0.5) * 20
    );
  }
  
  particles.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3));
  particles.userData.velocities = velocities;
  
  const material = new THREE.PointsMaterial({
    color: color,
    size: 0.5,
    transparent: true,
    opacity: 1.0
  });
  
  const explosion = new THREE.Points(particles, material);
  explosion.userData.startTime = Date.now();
  explosion.userData.duration = 2000; // 2 seconds
  
  scene.add(explosion);
  
  // Auto cleanup after animation
  setTimeout(() => {
    scene.remove(explosion);
    disposeObject(explosion);
  }, explosion.userData.duration);
  
  return explosion;
};

/**
 * Update explosion animation
 * @param {THREE.Points} explosion - Explosion particle system
 * @param {number} time - Current time
 */
export const updateExplosion = (explosion, time) => {
  if (!explosion.userData.velocities) return;
  
  const elapsed = time - explosion.userData.startTime;
  const progress = elapsed / explosion.userData.duration;
  
  if (progress >= 1) return;
  
  const positions = explosion.geometry.attributes.position.array;
  const velocities = explosion.userData.velocities;
  
  for (let i = 0; i < positions.length; i += 3) {
    positions[i] += velocities[i] * 0.016; // 60fps
    positions[i + 1] += velocities[i + 1] * 0.016;
    positions[i + 2] += velocities[i + 2] * 0.016;
    
    // Apply gravity
    velocities[i + 1] -= 0.5;
  }
  
  explosion.geometry.attributes.position.needsUpdate = true;
  
  // Fade out
  explosion.material.opacity = 1 - progress;
};

/**
 * Performance monitoring utilities
 */
export const PerformanceMonitor = {
  frameCount: 0,
  lastTime: 0,
  fps: 0,
  
  update() {
    this.frameCount++;
    const now = performance.now();
    
    if (now >= this.lastTime + 1000) {
      this.fps = Math.round((this.frameCount * 1000) / (now - this.lastTime));
      this.frameCount = 0;
      this.lastTime = now;
    }
  },
  
  getFPS() {
    return this.fps;
  },
  
  shouldReduceQuality() {
    return this.fps < 30;
  }
};