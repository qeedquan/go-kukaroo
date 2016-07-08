#!/usr/bin/env python
import pygame
from pygame.locals import *
import sys
from random import randint, choice
import os
pygame.init()
screen = pygame.display.set_mode((660, 390), 0, 32)
os.chdir(os.path.abspath(os.path.dirname(sys.argv[0])+"/Content"))
clock = pygame.time.Clock()
pygame.display.set_caption("Kukaroo!")
flapsound = pygame.mixer.Sound('Flap.wav')
flapsound.set_volume(.2)
pygame.mixer.music.load('Music.mp3')


walls = []
borders = []
enemies = []

class Background():
	def __init__(self, pos, image):
		self.image = pygame.image.load(image).convert()
		self.x = pos[0]
		self.y = pos[1]
	def blitter(self):
		screen.blit(self.image, (self.x, self.y))
		
bg = Background((-50, -50), 'BG.jpg')
intro = Background((0, 0), 'Intro.png')

def getRGB():
	levelIMG = pygame.image.load('Level%d.png'%player.in_level).convert_alpha()
	del walls[:], feathers[:], borders[:], enemies[:]
	for x in range(levelIMG.get_width()):
		for y in range(levelIMG.get_height()):
			color = levelIMG.get_at((x, y))
			if color == (255, 255, 0, 255):
				enemies.append(SawBlade((x*30-30, y*30-30)))
			elif color == (0, 0, 255, 255):
				enemies.append(ElectricBox((x*30-30, y*30-30)))
			elif color == (0, 255, 255, 255):
				enemies.append(FallingBlock((x*30-30, y*30-30)))
			elif color == (255, 255, 255, 255):
				walls.append(Wall((x*30-30, y*30-30)))
			elif color == (255, 0, 255, 255):
				walls.append(Button((x*30-30, y*30-30)))
			elif color == (255, 0, 0, 255): #Red tiles reduce levelnum by 1
				borders.append(Border((x*30-30, y*30-30), -1))
			elif color == (0, 255, 0, 255): # Green tiles increase levelnum by 
				borders.append(Border((x*30-30, y*30-30), 1))

class Block():
	def __init__(self, pos):
		self.x = pos[0]
		self.y = pos[1]
		self.dx = self.dy = 0
		self.rect = pygame.Rect(self.x, self.y, 28, 28)
		self.degrees = 0
	def render(self):
		pygame.draw.rect(screen, (0, 0, 0), self.rect)
	def blitter(self):
		if self.degrees != 0:
			screen.blit(pygame.transform.rotate(self.image, self.degrees), (self.x, self.y))
		else:
			screen.blit(self.image, (self.x, self.y))
	def updateRect(self):
		self.rect.x, self.rect.y = self.x, self.y

class Wall(Block):
	def __init__(self, pos):
		Block.__init__(self, pos)
		self.image = pygame.image.load('Crate.png').convert_alpha()
class Button(Block):
	def __init__(self, pos):
		Block.__init__(self, pos)
		self.image = pygame.image.load('Button.png').convert_alpha()		
class Border(Block):
	def __init__(self, pos, change_level):
		Block.__init__(self, pos)
		self.change_level = change_level

		
class SawBlade(Block):
	def __init__(self, pos):
		Block.__init__(self, pos)
		self.num = 0
		self.IMG0 = pygame.image.load('Sawblade0.png').convert_alpha()
		self.IMG1 = pygame.image.load('Sawblade1.png').convert_alpha()
		self.image = self.IMG0
		self.dx = choice([-2, 2])
	def move(self):
		self.x += self.dx
		if self.num == 0:
			self.image = self.IMG0
		elif self.num == 1:
			self.image = self.IMG1
		self.num += 1
		if self.num > 1:
			self.num = 0
		self.updateRect()
		for wall in walls:
			if self.rect.colliderect(wall.rect):
				self.dx = -self.dx
				self.x += self.dx
class ElectricBox(Block):
	def __init__(self, pos):
		Block.__init__(self, pos)
		self.num = 0
		self.IMG0 = pygame.image.load('Electric0.png').convert_alpha()
		self.IMG1 = pygame.image.load('Electric1.png').convert_alpha()
		self.image = self.IMG0
	def move(self):
		if self.num >= 4:
			self.image = self.IMG0
		elif self.num < 4:
			self.image = self.IMG1
		self.num += 1
		if self.num > 8:
			self.num = 0
class FallingBlock(Block):
	def __init__(self, pos):
		Block.__init__(self, pos)
		self.image = pygame.image.load('Crate.png').convert_alpha()
		self.dy = 0
	def move(self):
		self.y += self.dy
		self.updateRect()
		if self.x - player.x < 40:
			self.dy = 9

class Player(Block):
	def __init__(self, pos):
		Block.__init__(self, pos)
		self.orig_y = self.y
		self.degrees = 0
		self.wingdownIMG = pygame.image.load('Canary0.png').convert_alpha()
		self.wingupIMG = pygame.image.load('Canary1.png').convert_alpha()
		self.image = self.wingdownIMG
		self.in_level = 1
		self.in_game = False
		self.last_pos = (self.x, self.y)
	def move(self):
		self.x += self.dx
		self.updateRect()
		for wall in walls:
			if wall.rect.colliderect(self.rect):
				if self.dx > 0:
					self.rect.right = wall.rect.left
				if self.dx < 0:
					self.rect.left = wall.rect.right
				self.x = self.rect.x
				self.updateRect()
		self.gravity()
	def returnToLast(self):
		self.x, self.y = self.last_pos[0], self.last_pos[1]
		self.updateRect()
	def blitter(self):
		if self.degrees != 0:
			screen.blit(pygame.transform.rotate(self.image, self.degrees), (self.x-5, self.y-2))
		else:
			screen.blit(self.image, (self.x-5, self.y))
	def gravity(self):
		self.dy += .08 # Continuously fall faster
		self.y += self.dy
		self.updateRect()
		if self.dy < 0:
			self.degrees = 0
			if self.dx > 0:
				self.image = self.wingdownIMG
			else:
				self.image = pygame.transform.flip(self.wingdownIMG, 1, 0)
			for wall in walls:
				if wall.rect.colliderect(self.rect):
					self.dy = 0
					self.rect.top = wall.rect.bottom
					self.y = self.rect.y
					self.updateRect()
		if self.dy >= 0:
			# Have the sprite slowly tilt downwards as the player falls
			if self.dx > 0:
				self.degrees -= .7
				self.image = self.wingupIMG
			else:
				self.degrees += .7
				self.image = pygame.transform.flip(self.wingupIMG, 1, 0)
			# When you land on a wall, stop falling
			for wall in walls:
				if wall.rect.colliderect(self.rect):
					self.rect.bottom = wall.rect.top
					self.y = self.rect.y
					self.updateRect()
					self.dy = 0
					self.degrees = 0
			
				
		
class DroppedFeather(Block):
	def __init__(self, pos):
		Block.__init__(self, pos)
		self.orig_y = self.y
		self.orig_x = self.x
		self.dy = randint(2, 5) * .2
		self.dx = randint(2, 4)* .3
		self.direction = choice([-1, 1])
		self.max_dist = randint(10, 45) # How far a feather can move horizontally
		self.time = 250
		self.image = pygame.image.load('Feather.png').convert_alpha()
		self.degrees = 0
	def fall(self):
		if self.direction == -1:
			self.x -= self.dx
			self.degrees += .5
			if self.x < self.orig_x - self.max_dist:
				self.direction = 1
		elif self.direction == 1:
			self.x += self.dx
			self.degrees -= .5
			if self.x > self.orig_x + self.max_dist:
				self.direction = -1
		self.y += self.dy
		self.updateRect()
		self.time -= 1

feathers = []
		
player = Player((50, 50))

getRGB()

pygame.mixer.music.play(-1)

def outGame():
	screen.fill((0, 0, 0))
	intro.blitter()
	for event in pygame.event.get():
		if event.type == QUIT:
			pygame.quit()
			sys.exit()
		if event.type == KEYDOWN:
			if event.key == ord(' '):
				player.in_game = True
			if event.key == K_ESCAPE:
				pygame.quit()
				sys.exit()

def inGame():
	screen.fill((250, 250, 250))
	for event in pygame.event.get():
		if event.type == QUIT:
			pygame.quit()
			sys.exit()
		if event.type == KEYDOWN:
			if event.key == K_ESCAPE:
				pygame.quit()
				sys.exit()
			if event.key == K_a:
				player.dx = -2
			if event.key == K_d:
				player.dx = 2
			if event.key == K_n:
				getRGB()
			if event.key == K_p:
				player.in_game = False			
			if event.key == K_SPACE:
				flapsound.play()
				player.dy = -2.5
				# Drop feathers with each flap of the wings
				for i in range(randint(2,4)):
					feathers.append(DroppedFeather((player.x+randint(-5, 5), player.y+randint(0, 7))))
		if event.type == KEYUP:
			if event.key == K_d or event.key == K_a: player.dx *= .5
	bg.blitter()
	for wall in walls:
		wall.blitter()
	for feather in feathers:
		feather.fall()
		feather.blitter()
		if feather.time <= 0:
			feathers.remove(feather)
	for enemy in enemies:
		enemy.move()
		enemy.blitter()	
		if enemy.rect.colliderect(player.rect): # Restart when hitting an enemy
			player.returnToLast()
			player.dy = 0
			getRGB()
	
	player.move()
	player.blitter()
	
	# The border tiles. Touch one and go to the next part of the map
	for border in borders:
		if player.rect.colliderect(border.rect):
			player.in_level += border.change_level
			if player.x > 630:
				player.x = 5
				bg.x -= 20
			elif player.x < 10:
				player.x = 610
				bg.x += 20
			elif player.y > 350:
				player.y = 5
				bg.y -= 20
			elif player.y < 10:
				player.y = player.y = 360
				bg.y += 20
			else: pass
			player.last_pos = (player.x, player.y)
			player.updateRect()
			getRGB()
	if player.in_level > 20:
		intro.image = pygame.image.load('Finish.jpg').convert()
		player.in_game = False
	clock.tick(60)
while True:
	if player.in_game == False:
		outGame()
	else:
		inGame()
	pygame.display.update()